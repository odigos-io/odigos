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

// languageCoverage joins each container's detected language (status) with the
// instrumentor's agent decision (spec). Counting by language rather than by
// distro name keeps a container that landed on a fallback distro (e.g.
// nodejs-community-14) on its language's row.
func languageCoverage(ctx context.Context, k8sCacheClient client.Client) (map[common.ProgrammingLanguage]*containerCoverage, error) {
	var instrumentationConfigs odigosv1.InstrumentationConfigList
	if err := k8sCacheClient.List(ctx, &instrumentationConfigs); err != nil {
		return nil, fmt.Errorf("listing instrumentation configs: %w", err)
	}

	coverage := make(map[common.ProgrammingLanguage]*containerCoverage)
	for i := range instrumentationConfigs.Items {
		ic := &instrumentationConfigs.Items[i]

		agentEnabled := make(map[string]bool, len(ic.Spec.Containers))
		for _, container := range ic.Spec.Containers {
			agentEnabled[container.ContainerName] = container.AgentEnabled
		}

		// A source running two containers of the same language is still one
		// source, so the language is tallied after all of its containers.
		languagesInSource := make(map[common.ProgrammingLanguage]bool)

		for _, runtimeDetails := range ic.Status.RuntimeDetailsByContainer {
			if runtimeDetails.Language == "" {
				continue
			}
			counts := coverage[runtimeDetails.Language]
			if counts == nil {
				counts = &containerCoverage{}
				coverage[runtimeDetails.Language] = counts
			}
			if agentEnabled[runtimeDetails.ContainerName] {
				counts.instrumented++
			} else {
				counts.uninstrumented++
			}
			languagesInSource[runtimeDetails.Language] = true
		}

		for language := range languagesInSource {
			coverage[language].sources++
		}
	}
	return coverage, nil
}

// fallbackDistroNames walks the fallbackDistro chain from the given distro,
// newest to oldest. The seen set guards against a cycle in the yaml.
func fallbackDistroNames(getter *distros.Getter, distroName string) []string {
	names := make([]string, 0)
	seen := map[string]bool{}
	current := distroName
	for {
		d := getter.GetDistroByName(current)
		if d == nil || d.FallbackDistro == nil || seen[current] {
			return names
		}
		seen[current] = true
		current = *d.FallbackDistro
		names = append(names, current)
	}
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

// GetInstrumentationAgents returns one entry per language that has a default
// distro in the running tier, annotated with this cluster's coverage. The
// provider is the same one the instrumentor resolves distros with, so the page
// reflects what will actually be injected rather than a parallel guess.
func GetInstrumentationAgents(ctx context.Context, k8sCacheClient client.Client, provider *distros.Provider) ([]*model.InstrumentationAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("no distros provider configured")
	}

	communityNames, err := getCommunityDistroNames()
	if err != nil {
		return nil, fmt.Errorf("loading community distros: %w", err)
	}

	coverage, err := languageCoverage(ctx, k8sCacheClient)
	if err != nil {
		return nil, err
	}

	defaults := provider.GetDefaultDistroNames()
	agents := make([]*model.InstrumentationAgent, 0, len(defaults))
	for language, distroName := range defaults {
		d := provider.GetDistroByName(distroName)
		if d == nil {
			// NewProvider rejects a defaulter naming a distro no getter has, so
			// this is unreachable; skip rather than fail the whole page.
			continue
		}

		tier := model.InstrumentationAgentTierEnterprise
		if communityNames[d.Name] {
			tier = model.InstrumentationAgentTierCommunity
		}

		agent := &model.InstrumentationAgent{
			Language:            string(language),
			DistroName:          d.Name,
			DistroDisplayName:   d.DisplayName,
			Description:         strings.TrimSpace(d.Description),
			Tier:                tier,
			Kind:                agentKind(d),
			FallbackDistroNames: fallbackDistroNames(provider.Getter, d.Name),
		}
		if len(d.RuntimeEnvironments) > 0 {
			agent.RuntimeEnvironment = d.RuntimeEnvironments[0].Name
			agent.SupportedRuntimeVersions = d.RuntimeEnvironments[0].SupportedVersions
		}
		if counts := coverage[language]; counts != nil {
			agent.InstrumentedContainers = counts.instrumented
			agent.UninstrumentedContainers = counts.uninstrumented
			agent.Sources = counts.sources
		}

		agents = append(agents, agent)
	}

	sort.Slice(agents, func(i, j int) bool { return agents[i].Language < agents[j].Language })
	return agents, nil
}
