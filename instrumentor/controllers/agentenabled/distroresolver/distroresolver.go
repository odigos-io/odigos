package distroresolver

import (
	"fmt"

	"github.com/hashicorp/go-version"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
)

func resolveDistroByOverride(overwriteDistroName string, distroGetter *distros.Getter, containerLanguage common.ProgrammingLanguage) (*distro.OtelDistro, *odigosv1.AgentDisabledInfo) {
	distro := distroGetter.GetDistroByName(overwriteDistroName)
	if distro == nil { // not expected to happen, here for safety net
		message := fmt.Sprintf("requested otel distro %s is not available in this odigos tier", overwriteDistroName)
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
			AgentEnabledMessage: message,
		}
	}

	// Wildcard-language distros skip container language match when explicitly selected.
	if common.IsProgrammingLanguageWildcard(distro.Language) {
		return distro, nil
	}

	// verify the distro matches the language, since it might be overridden by the container override.
	if distro.Language != containerLanguage {
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage,
			AgentEnabledMessage: fmt.Sprintf("requested otel distro %s does not support language %s", overwriteDistroName, containerLanguage),
		}
	}

	return distro, nil
}

func resolveDistroByLanguage(containerLanguage common.ProgrammingLanguage, distroPerLanguage map[common.ProgrammingLanguage]string, distroGetter *distros.Getter, runtimeVersion string) (*distro.OtelDistro, *odigosv1.AgentDisabledInfo) {
	defaultDistroName, ok := distroPerLanguage[containerLanguage]
	if !ok {
		if containerLanguage == common.UnknownProgrammingLanguage {
			return nil, &odigosv1.AgentDisabledInfo{
				AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
				AgentEnabledMessage: "runtime language/platform cannot be detected, no instrumentation agent is available. use the container override to manually specify the programming language.",
			}
		} else {
			return nil, &odigosv1.AgentDisabledInfo{
				AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
				AgentEnabledMessage: fmt.Sprintf("support for %s is coming soon. no instrumentation agent available at the moment", containerLanguage),
			}
		}
	}

	// Walk the fallbackDistro chain to resolve the best-matching distro name for the runtime version.
	effectiveDistroName := distroGetter.ResolveDistroNameForVersion(defaultDistroName, runtimeVersion)

	distro := distroGetter.GetDistroByName(effectiveDistroName)
	if distro == nil { // not expected to happen, here for safety net
		message := fmt.Sprintf("otel distro %s is not available in this odigos tier", effectiveDistroName)
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
			AgentEnabledMessage: message,
		}
	}

	return distro, nil
}

func CalculateDefaultDistroPerLanguage(defaultDistros map[common.ProgrammingLanguage]string,
	instrumentationRules *[]odigosv1.InstrumentationRule, dg *distros.Getter,
) map[common.ProgrammingLanguage]string {
	distrosPerLanguage := make(map[common.ProgrammingLanguage]string, len(defaultDistros))
	for lang, distroName := range defaultDistros {
		distrosPerLanguage[lang] = distroName
	}

	for _, rule := range *instrumentationRules {
		if rule.Spec.OtelDistros == nil {
			continue
		}
		for _, distroName := range rule.Spec.OtelDistros.OtelDistroNames {
			distro := dg.GetDistroByName(distroName)
			if distro == nil {
				continue
			}
			if common.IsProgrammingLanguageWildcard(distro.Language) {
				continue
			}

			lang := distro.Language
			distrosPerLanguage[lang] = distroName
		}
	}

	return distrosPerLanguage
}

func ResolveDistroForContainer(
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	distroPerLanguage map[common.ProgrammingLanguage]string,
	distroGetter *distros.Getter,
	containerOverride *odigosv1.ContainerOverride,
	containerName string,
) (*distro.OtelDistro, *odigosv1.AgentDisabledInfo) {
	// check if container is ignored by name, assuming IgnoredContainers is a short list.
	// This should be done first, because user should see workload not instrumented if container is ignored over unknown language in case both exist.
	for _, ignoredContainer := range effectiveConfig.IgnoredContainers {
		if ignoredContainer == containerName {
			return nil, &odigosv1.AgentDisabledInfo{
				AgentEnabledReason:  odigosv1.AgentEnabledReasonIgnoredContainer,
				AgentEnabledMessage: "container is ignored",
			}
		}
	}

	if runtimeDetails == nil {
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonRuntimeDetailsUnavailable,
			AgentEnabledMessage: "runtime details are unavailable",
		}
	}

	// check unknown language first. if language is not supported, we can skip the rest of the checks.
	if runtimeDetails.Language == common.UnknownProgrammingLanguage {
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage,
			AgentEnabledMessage: "unknown programming language",
		}
	}

	// if the user specifically overwritten the distro to use for this container, use it.
	if containerOverride != nil && containerOverride.OtelDistroName != nil {
		return resolveDistroByOverride(*containerOverride.OtelDistroName, distroGetter, runtimeDetails.Language)
	}

	// use the default distro and detected language to resolve the distro to use.
	d, disabledInfo := resolveDistroByLanguage(runtimeDetails.Language, distroPerLanguage, distroGetter, runtimeDetails.RuntimeVersion)
	if disabledInfo != nil {
		return nil, disabledInfo
	}

	// check if the runtime version is in supported range if it is provided.
	// Wildcard-language distros (e.g. OBI) skip semver checks; supportedVersions may be '*'.
	if runtimeDetails.RuntimeVersion != "" && len(d.RuntimeEnvironments) == 1 && !common.IsProgrammingLanguageWildcard(d.Language) {
		constraint, err := version.NewConstraint(d.RuntimeEnvironments[0].SupportedVersions)
		if err != nil {
			return nil, &odigosv1.AgentDisabledInfo{
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("failed to parse supported versions constraint: %s", d.RuntimeEnvironments[0].SupportedVersions),
			}
		}
		detectedVersion, err := version.NewVersion(runtimeDetails.RuntimeVersion)
		if err != nil {
			return nil, &odigosv1.AgentDisabledInfo{
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("failed to parse runtime version: %s", runtimeDetails.RuntimeVersion),
			}
		}
		if !constraint.Check(detectedVersion) {
			return nil, &odigosv1.AgentDisabledInfo{
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("%s runtime not supported by OpenTelemetry. supported versions: '%s', found: %s", d.RuntimeEnvironments[0].Name, constraint, detectedVersion),
			}
		}
	}

	return d, nil
}
