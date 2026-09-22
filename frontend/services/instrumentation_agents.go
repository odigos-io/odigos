package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// distroSourceCounts joins each container's detected language (status) with the
// distro the instrumentor actually picked for it (spec), and counts the sources
// each distro covers. The page lists more than one distro per language — on
// enterprise a language usually has both an enterprise and a community entry —
// so counting by language would report the same sources on every row.
//
// A container the instrumentor did not enable has no distro of its own, and one
// that landed on a distro the page does not list (a version fallback such as
// nodejs-community-14) would otherwise vanish from the totals. Both are counted
// against the language's default distro, which is the row that answers "what
// would this container run".
func distroSourceCounts(ctx context.Context, k8sCacheClient client.Client, defaults map[common.ProgrammingLanguage]string, listed map[string]bool) (map[string]int, error) {
	var instrumentationConfigs odigosv1.InstrumentationConfigList
	if err := k8sCacheClient.List(ctx, &instrumentationConfigs); err != nil {
		return nil, fmt.Errorf("listing instrumentation configs: %w", err)
	}

	sources := make(map[string]int)
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

			distrosInSource[distroName] = true
		}

		for distroName := range distrosInSource {
			sources[distroName]++
		}
	}
	return sources, nil
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
// instrument with, annotated with how many sources it covers. The provider is the
// same one the instrumentor resolves distros with, so the page reflects what
// will actually be injected rather than a parallel guess.
func GetInstrumentationAgents(ctx context.Context, k8sCacheClient client.Client, provider *distros.Provider) ([]*model.InstrumentationAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("no distros provider configured")
	}

	defaults := provider.GetDefaultDistroNames()
	distrosByLanguage := agentDistros(provider)

	listed := make(map[string]bool)
	for _, ds := range distrosByLanguage {
		for _, d := range ds {
			listed[d.Name] = true
		}
	}

	sourcesByDistro, err := distroSourceCounts(ctx, k8sCacheClient, defaults, listed)
	if err != nil {
		return nil, err
	}

	agents := make([]*model.InstrumentationAgent, 0, len(listed))
	for language, ds := range distrosByLanguage {
		for _, d := range ds {
			agent := &model.InstrumentationAgent{
				Language:          string(language),
				DistroName:        d.Name,
				DistroDisplayName: d.DisplayName,
				Description:       strings.TrimSpace(d.Description),
			}
			if len(d.RuntimeEnvironments) > 0 {
				agent.RuntimeEnvironment = d.RuntimeEnvironments[0].Name
				agent.SupportedRuntimeVersions = d.RuntimeEnvironments[0].SupportedVersions
			}
			agent.Sources = sourcesByDistro[d.Name]

			agents = append(agents, agent)
		}
	}

	// Language first so a language's distros stay adjacent, then the distro
	// workloads actually get, then by name.
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
		return a.DistroName < b.DistroName
	})
	return agents, nil
}
