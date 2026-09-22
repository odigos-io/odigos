package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

var (
	communityDistroNamesOnce sync.Once
	communityDistroNames     map[string]bool
	communityDistroNamesErr  error
)

// getCommunityDistroNames returns the set of distro names shipped by the OSS
// distros module. A provider flattens all of its getters into one map, so this
// set is what tells apart a community distro from an enterprise one — on an
// enterprise cluster several languages (.NET, PHP, Ruby) still resolve to a
// community distro, and the UI surfaces that per row.
func getCommunityDistroNames() (map[string]bool, error) {
	communityDistroNamesOnce.Do(func() {
		getter, err := distros.NewCommunityGetter()
		if err != nil {
			communityDistroNamesErr = err
			return
		}
		names := make(map[string]bool)
		for _, d := range getter.GetAllDistros() {
			names[d.Name] = true
		}
		communityDistroNames = names
	})
	return communityDistroNames, communityDistroNamesErr
}

// containerCoverage counts, for a single language, how many containers in the
// cluster the instrumentor did and did not enable an agent for, and how many
// sources those containers belong to.
type containerCoverage struct {
	instrumented   int
	uninstrumented int
	sources        int
}

// distroCoverage joins each container's detected language (status) with the
// distro the instrumentor actually picked for it (spec), keyed by distro name.
// The page lists more than one distro per language — on enterprise a language
// usually has both an enterprise and a community entry — so counting by
// language would report the same containers on every row of that language.
//
// A container the instrumentor did not enable has no distro of its own, and one
// that landed on a distro the page does not list (a version fallback such as
// nodejs-community-14) would otherwise vanish from the totals. Both are counted
// against the language's default distro, which is the row that answers "what
// would this container run".
func distroCoverage(ctx context.Context, k8sCacheClient client.Client, defaults map[common.ProgrammingLanguage]string, listed map[string]bool) (map[string]*containerCoverage, error) {
	var instrumentationConfigs odigosv1.InstrumentationConfigList
	if err := k8sCacheClient.List(ctx, &instrumentationConfigs); err != nil {
		return nil, fmt.Errorf("listing instrumentation configs: %w", err)
	}

	coverage := make(map[string]*containerCoverage)
	for i := range instrumentationConfigs.Items {
		ic := &instrumentationConfigs.Items[i]

		specByContainer := make(map[string]odigosv1.ContainerAgentConfig, len(ic.Spec.Containers))
		for _, container := range ic.Spec.Containers {
			specByContainer[container.ContainerName] = container
		}

		// A source running two containers on the same distro is still one
		// source, so distros are tallied after all of the source's containers.
		distrosInSource := make(map[string]bool)

		for _, runtimeDetails := range ic.Status.RuntimeDetailsByContainer {
			if runtimeDetails.Language == "" {
				continue
			}
			spec := specByContainer[runtimeDetails.ContainerName]

			distroName := spec.OtelDistroName
			if !spec.AgentEnabled || !listed[distroName] {
				distroName = defaults[runtimeDetails.Language]
			}
			if distroName == "" {
				continue
			}

			counts := coverage[distroName]
			if counts == nil {
				counts = &containerCoverage{}
				coverage[distroName] = counts
			}
			if spec.AgentEnabled {
				counts.instrumented++
			} else {
				counts.uninstrumented++
			}
			distrosInSource[distroName] = true
		}

		for distroName := range distrosInSource {
			coverage[distroName].sources++
		}
	}
	return coverage, nil
}

// agentKind reports how the distro instruments the workload. Distros that
// combine eBPF with a mounted agent directory (java-ebpf-instrumentations,
// nodejs-enterprise) are still eBPF-based, so the distro's own declaration is
// what the UI reports.
func agentKind(d *distro.OtelDistro) model.InstrumentationAgentKind {
	if d.IsEbpf {
		return model.InstrumentationAgentKindEbpf
	}
	return model.InstrumentationAgentKindCodeAgent
}

// agentDistros returns the distros the page lists, grouped by language.
//
// A provider unions every getter it was built with, so on enterprise this is the
// community and enterprise sets together: an enterprise cluster still
// instruments several languages with a community distro, and both belong here.
// Each distro declares its own language, so the grouping needs no defaulter.
//
// Two kinds are left out. A distro that another one falls back to is only
// reached when a runtime version drifts out of range, making it a property of
// its parent rather than an agent in its own right — unless it is the tier's
// default, in which case it is exactly what workloads get. And a language the
// running tier has no default for is not instrumented on this cluster at all,
// which also drops distros that declare the wildcard language.
func agentDistros(provider *distros.Provider) map[common.ProgrammingLanguage][]*distro.OtelDistro {
	all := provider.GetAllDistros()

	fallbacks := make(map[string]bool, len(all))
	for _, d := range all {
		if d.FallbackDistro != nil {
			fallbacks[*d.FallbackDistro] = true
		}
	}

	defaults := provider.GetDefaultDistroNames()
	byLanguage := make(map[common.ProgrammingLanguage][]*distro.OtelDistro)
	for _, d := range all {
		defaultName, instrumented := defaults[d.Language]
		if !instrumented {
			continue
		}
		if fallbacks[d.Name] && d.Name != defaultName {
			continue
		}
		byLanguage[d.Language] = append(byLanguage[d.Language], d)
	}

	return byLanguage
}

// GetInstrumentationAgents returns one entry per distro the cluster can
// instrument with, annotated with this cluster's coverage. The provider is the
// same one the instrumentor resolves distros with, so the page reflects what
// will actually be injected rather than a parallel guess.
func GetInstrumentationAgents(ctx context.Context, k8sCacheClient client.Client, provider *distros.Provider) ([]*model.InstrumentationAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("no distros provider configured")
	}

	communityNames, err := getCommunityDistroNames()
	if err != nil {
		return nil, fmt.Errorf("loading community distros: %w", err)
	}

	defaults := provider.GetDefaultDistroNames()
	distrosByLanguage := agentDistros(provider)

	listed := make(map[string]bool)
	for _, ds := range distrosByLanguage {
		for _, d := range ds {
			listed[d.Name] = true
		}
	}

	coverage, err := distroCoverage(ctx, k8sCacheClient, defaults, listed)
	if err != nil {
		return nil, err
	}

	agents := make([]*model.InstrumentationAgent, 0, len(listed))
	for language, ds := range distrosByLanguage {
		for _, d := range ds {
			tier := model.InstrumentationAgentTierEnterprise
			if communityNames[d.Name] {
				tier = model.InstrumentationAgentTierCommunity
			}

			agent := &model.InstrumentationAgent{
				Language:          string(language),
				DistroName:        d.Name,
				DistroDisplayName: d.DisplayName,
				Description:       strings.TrimSpace(d.Description),
				Tier:              tier,
				Kind:              agentKind(d),
			}
			if len(d.RuntimeEnvironments) > 0 {
				agent.RuntimeEnvironment = d.RuntimeEnvironments[0].Name
				agent.SupportedRuntimeVersions = d.RuntimeEnvironments[0].SupportedVersions
			}
			if counts := coverage[d.Name]; counts != nil {
				agent.InstrumentedContainers = counts.instrumented
				agent.UninstrumentedContainers = counts.uninstrumented
				agent.Sources = counts.sources
			}

			agents = append(agents, agent)
		}
	}

	// Language first so a language's distros stay adjacent, then the distro
	// workloads actually get, then enterprise ahead of community.
	isDefault := func(a *model.InstrumentationAgent) bool {
		return defaults[common.ProgrammingLanguage(a.Language)] == a.DistroName
	}
	sort.Slice(agents, func(i, j int) bool {
		a, b := agents[i], agents[j]
		if a.Language != b.Language {
			return a.Language < b.Language
		}
		if isDefault(a) != isDefault(b) {
			return isDefault(a)
		}
		if a.Tier != b.Tier {
			return a.Tier == model.InstrumentationAgentTierEnterprise
		}
		return a.DistroName < b.DistroName
	})
	return agents, nil
}
