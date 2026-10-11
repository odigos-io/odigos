package profiles

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/manifests"
	"github.com/odigos-io/odigos/profiles/profile"
)

func prNameSet(ps []profile.Profile) map[common.ProfileName]bool {
	set := make(map[common.ProfileName]bool, len(ps))
	for _, p := range ps {
		set[p.ProfileName] = true
	}
	return set
}

func prSortedNames(ps []profile.Profile) []string {
	names := make([]string, 0, len(ps))
	for _, p := range ps {
		names = append(names, string(p.ProfileName))
	}
	slices.Sort(names)
	return names
}

// TestProfilesByNameCoversEveryProfileExactlyOnce guards the init() registry. The map is
// built by assignment, so two profiles declared with the same ProfileName silently shadow
// each other: only the last one is ever reachable by name, and the shadowed one's manifest
// and ModifyConfigFunc are never applied without any error being raised.
func TestProfilesByNameCoversEveryProfileExactlyOnce(t *testing.T) {
	if len(AllProfiles) == 0 {
		t.Fatal("AllProfiles is empty")
	}
	if len(ProfilesByName) != len(AllProfiles) {
		t.Errorf("ProfilesByName has %d entries for %d profiles, which means a duplicate ProfileName shadowed another profile; names: %v",
			len(ProfilesByName), len(AllProfiles), prSortedNames(AllProfiles))
	}
	for _, p := range AllProfiles {
		if p.ProfileName == "" {
			t.Error("found a profile with an empty ProfileName")
			continue
		}
		byName, ok := ProfilesByName[p.ProfileName]
		if !ok {
			t.Errorf("profile %q is in AllProfiles but missing from ProfilesByName", p.ProfileName)
			continue
		}
		if byName.ProfileName != p.ProfileName || byName.MinimumTier != p.MinimumTier {
			t.Errorf("ProfilesByName[%q] = {name:%q tier:%q}, want {name:%q tier:%q}",
				p.ProfileName, byName.ProfileName, byName.MinimumTier, p.ProfileName, p.MinimumTier)
		}
	}
}

// TestEveryProfileIsAvailableOnPrem pins that the tier fan-out in init() recognizes the
// MinimumTier of every declared profile. The switch has no default branch, so a profile
// carrying any other tier lands in neither CommunityProfiles nor OnPremProfiles and becomes
// invisible to every tier - it can never be enabled and nothing reports it.
func TestEveryProfileIsAvailableOnPrem(t *testing.T) {
	onPrem := prNameSet(OnPremProfiles)
	for _, p := range AllProfiles {
		if !onPrem[p.ProfileName] {
			t.Errorf("profile %q (MinimumTier %q) is not available to any tier; init() only fans out community and onprem",
				p.ProfileName, p.MinimumTier)
		}
	}
	if len(OnPremProfiles) != len(AllProfiles) {
		t.Errorf("OnPremProfiles has %d of %d profiles", len(OnPremProfiles), len(AllProfiles))
	}
}

// TestCommunityProfilesAreASubsetOfOnPrem pins the "community profiles are also on-prem
// profiles" rule from init(). If it regressed, upgrading a cluster from community to
// on-prem would silently drop the profiles it already had enabled.
func TestCommunityProfilesAreASubsetOfOnPrem(t *testing.T) {
	if len(CommunityProfiles) == 0 {
		t.Fatal("CommunityProfiles is empty")
	}
	if len(CommunityProfiles) >= len(OnPremProfiles) {
		t.Fatalf("expected strictly fewer community profiles (%d) than on-prem profiles (%d); otherwise the tier gate is not distinguishing anything",
			len(CommunityProfiles), len(OnPremProfiles))
	}

	onPrem := prNameSet(OnPremProfiles)
	for _, p := range CommunityProfiles {
		if p.MinimumTier != common.CommunityOdigosTier {
			t.Errorf("profile %q is in CommunityProfiles but its MinimumTier is %q", p.ProfileName, p.MinimumTier)
		}
		if !onPrem[p.ProfileName] {
			t.Errorf("community profile %q is missing from OnPremProfiles", p.ProfileName)
		}
	}
}

// TestGetAvailableProfilesForTier drives the tier gate the scheduler uses to decide which
// profiles a cluster may enable, including its transitive dependencies. The unrecognized-tier
// cases are pinned deliberately: an unset or unknown tier must resolve to no profiles at all
// rather than to the full on-prem set, so the entitlement check fails closed. CloudOdigosTier
// is a declared tier that this function does not recognize today - it is asserted here so that
// teaching the gate about it is a visible, deliberate change rather than an accident.
func TestGetAvailableProfilesForTier(t *testing.T) {
	community := GetAvailableProfilesForTier(common.CommunityOdigosTier)
	if got, want := prSortedNames(community), prSortedNames(CommunityProfiles); !slices.Equal(got, want) {
		t.Errorf("community tier profiles = %v, want %v", got, want)
	}

	onPrem := GetAvailableProfilesForTier(common.OnPremOdigosTier)
	if got, want := prSortedNames(onPrem), prSortedNames(AllProfiles); !slices.Equal(got, want) {
		t.Errorf("onprem tier profiles = %v, want %v", got, want)
	}

	for _, tier := range []common.OdigosTier{
		common.CloudOdigosTier,
		common.OdigosTier(""),
		common.OdigosTier("onprem "),
		common.OdigosTier("OnPrem"),
		common.OdigosTier("enterprise"),
	} {
		got := GetAvailableProfilesForTier(tier)
		if got == nil {
			t.Errorf("GetAvailableProfilesForTier(%q) returned a nil slice, callers range over it and expect empty", string(tier))
		}
		if len(got) != 0 {
			t.Errorf("GetAvailableProfilesForTier(%q) = %v, want no profiles for an unrecognized tier", string(tier), prSortedNames(got))
		}
	}
}

// TestEveryProfileDependencyResolvesToADeclaredProfile guards the aggregator profiles
// (kratos, greatwall, insights), which are nothing but a list of dependency names. The
// scheduler resolves those names with a lookup that silently skips anything it cannot find,
// so a typo or a rename of a dependency removes a preset from every cluster with no error,
// no log line and no change to the profile's description.
func TestEveryProfileDependencyResolvesToADeclaredProfile(t *testing.T) {
	checked := 0
	for _, p := range AllProfiles {
		for _, dep := range p.Dependencies {
			checked++
			if _, ok := ProfilesByName[dep]; !ok {
				t.Errorf("profile %q depends on %q, which is not a declared profile; the scheduler will silently drop it",
					p.ProfileName, dep)
			}
			if dep == p.ProfileName {
				t.Errorf("profile %q depends on itself", p.ProfileName)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no dependencies were checked, the test is vacuous")
	}
}

// TestProfileDependenciesDoNotRequireAHigherTier pins the entitlement invariant that makes
// aggregator profiles work: a profile must not depend on a profile that requires a higher
// tier than it does. The scheduler resolves dependencies against the tier-filtered profile
// list, so a dependency above the parent's tier is dropped and the parent silently applies
// only part of what it promises - worst of all for the aggregators, which have no manifest
// and no ModifyConfigFunc of their own and therefore degrade to a complete no-op.
//
// This is exactly the state the insights profile was in while it was declared community tier
// and depended on the on-prem full-payload-collection profile.
func TestProfileDependenciesDoNotRequireAHigherTier(t *testing.T) {
	checked := 0
	for _, p := range AllProfiles {
		availableToParent := prNameSet(GetAvailableProfilesForTier(p.MinimumTier))
		for _, dep := range p.Dependencies {
			depProfile, ok := ProfilesByName[dep]
			if !ok {
				continue // reported by TestEveryProfileDependencyResolvesToADeclaredProfile
			}
			checked++
			if !availableToParent[dep] {
				t.Errorf("profile %q (tier %q) depends on %q (tier %q), which is not available at the parent's tier: on that tier the dependency is silently dropped",
					p.ProfileName, p.MinimumTier, dep, depProfile.MinimumTier)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no dependencies were checked, the test is vacuous")
	}
}

// prFindDependencyCycle returns a cycle in the dependency graph, or nil if it is acyclic.
func prFindDependencyCycle(graph map[common.ProfileName][]common.ProfileName) []common.ProfileName {
	// The zero value means "not visited yet".
	const (
		onStack = 1
		done    = 2
	)
	state := make(map[common.ProfileName]int, len(graph))
	var stack []common.ProfileName

	var visit func(common.ProfileName) []common.ProfileName
	visit = func(name common.ProfileName) []common.ProfileName {
		switch state[name] {
		case onStack:
			cycle := slices.Clone(stack[slices.Index(stack, name):])
			return append(cycle, name)
		case done:
			return nil
		}
		state[name] = onStack
		stack = append(stack, name)
		for _, dep := range graph[name] {
			if cycle := visit(dep); cycle != nil {
				return cycle
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = done
		return nil
	}

	names := make([]string, 0, len(graph))
	for name := range graph {
		names = append(names, string(name))
	}
	slices.Sort(names)
	for _, name := range names {
		if cycle := visit(common.ProfileName(name)); cycle != nil {
			return cycle
		}
	}
	return nil
}

// TestProfileDependencyGraphIsAcyclic is the guard for the most damaging mistake this
// registry can express. The scheduler expands dependencies with an unbounded recursion that
// keeps no visited set, so any cycle - including a one-element self-dependency - makes the
// expansion recurse until the goroutine stack limit is hit. That is a `fatal error: stack
// overflow`, not a returned error: the scheduler process dies, restarts, reconciles the same
// configuration and dies again, so no effective configuration is ever written and the whole
// cluster is left without a working pipeline configuration.
//
// The cycle is only ever introduced here, in the declared Dependencies, which is why the
// guard lives in this package.
func TestProfileDependencyGraphIsAcyclic(t *testing.T) {
	graph := make(map[common.ProfileName][]common.ProfileName, len(AllProfiles))
	for _, p := range AllProfiles {
		graph[p.ProfileName] = p.Dependencies
	}

	// Prove the detector can actually see a cycle before trusting it on the real graph.
	for _, tc := range []struct {
		name  string
		graph map[common.ProfileName][]common.ProfileName
	}{
		{
			name:  "self dependency",
			graph: map[common.ProfileName][]common.ProfileName{"a": {"a"}},
		},
		{
			name:  "two element cycle",
			graph: map[common.ProfileName][]common.ProfileName{"a": {"b"}, "b": {"a"}},
		},
		{
			name:  "cycle reachable only through a longer chain",
			graph: map[common.ProfileName][]common.ProfileName{"a": {"b"}, "b": {"c"}, "c": {"d"}, "d": {"b"}},
		},
	} {
		if cycle := prFindDependencyCycle(tc.graph); cycle == nil {
			t.Fatalf("the cycle detector missed the %s fixture, so it cannot be trusted on the real graph", tc.name)
		}
	}
	// A diamond is not a cycle: the same dependency reached twice must not be reported.
	if cycle := prFindDependencyCycle(map[common.ProfileName][]common.ProfileName{
		"a": {"b", "c"}, "b": {"d"}, "c": {"d"}, "d": nil,
	}); cycle != nil {
		t.Fatalf("the cycle detector reported %v for an acyclic diamond", cycle)
	}

	if cycle := prFindDependencyCycle(graph); cycle != nil {
		t.Errorf("the profile dependency graph has a cycle %v; the scheduler expands dependencies recursively with no visited set and will crash with a stack overflow", cycle)
	}
}

// prDeprecatedNoOpProfiles are profiles that are intentionally inert: they are kept so that
// clusters listing them by name keep reconciling, but the behavior they used to enable is now
// the default. Any other inert profile is a mistake, so this list is only ever consulted to
// excuse a profile from TestEveryProfileHasAnEffect - being listed here does not require a
// profile to stay inert.
var prDeprecatedNoOpProfiles = map[common.ProfileName]bool{
	"java-native-instrumentations": true,
}

// TestEveryProfileHasAnEffect pins the coupling between a profile's name and the way it takes
// effect. A profile can only do something through an embedded manifest, a ModifyConfigFunc or
// its dependencies. The manifest is looked up by filename derived from the profile name and a
// missing file is not an error, so renaming a profile without renaming its yaml - or renaming
// the yaml without renaming the profile - turns the profile into a silent no-op. Nothing
// reports it: the profile still appears in the CLI listing and in the effective configuration,
// it simply stops applying its rule.
func TestEveryProfileHasAnEffect(t *testing.T) {
	for _, p := range AllProfiles {
		yamls, err := manifests.ReadProfileYamlManifests(p.ProfileName)
		if err != nil {
			t.Errorf("reading manifests for %q: %v", p.ProfileName, err)
			continue
		}
		if len(yamls) > 0 || p.ModifyConfigFunc != nil || len(p.Dependencies) > 0 {
			continue
		}
		if prDeprecatedNoOpProfiles[p.ProfileName] {
			continue
		}
		t.Errorf("profile %q has no manifest %q.yaml, no ModifyConfigFunc and no dependencies, so enabling it does nothing",
			p.ProfileName, p.ProfileName)
	}
}

// TestEveryEmbeddedManifestBelongsToADeclaredProfile is the other direction of the same
// coupling. A manifest is only ever read through a lookup by profile name, so a yaml file whose
// basename is not a declared profile can never be applied to any cluster - it looks like a
// shipped rule while being dead weight. This is what a rename leaves behind when the profile
// declaration is updated and the file is not.
func TestEveryEmbeddedManifestBelongsToADeclaredProfile(t *testing.T) {
	entries, err := os.ReadDir("manifests")
	if err != nil {
		t.Fatalf("listing the manifests directory: %v", err)
	}

	found := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		found++
		profileName := common.ProfileName(strings.TrimSuffix(name, ".yaml"))
		if _, ok := ProfilesByName[profileName]; !ok {
			t.Errorf("manifests/%s does not belong to any declared profile, so it is never applied", name)
		}
	}
	if found == 0 {
		t.Fatal("no manifest files were found, the test is vacuous")
	}
}

// TestInsightsProfileGatesPayloadCollectionBehindOnPrem pins the two properties that make the
// insights bundle safe to enable. It pulls in full payload collection, which captures HTTP
// request and response bodies, database queries and messaging payloads from instrumented
// workloads - so it must stay an on-prem profile, both because payload collection itself is an
// on-prem feature and because the dependency would otherwise be dropped on lower tiers and
// leave the bundle half applied.
func TestInsightsProfileGatesPayloadCollectionBehindOnPrem(t *testing.T) {
	insights, ok := ProfilesByName["insights"]
	if !ok {
		t.Fatal("the insights profile is no longer declared")
	}
	if insights.MinimumTier != common.OnPremOdigosTier {
		t.Errorf("insights MinimumTier = %q, want %q: it depends on payload collection", insights.MinimumTier, common.OnPremOdigosTier)
	}
	if !slices.Contains(insights.Dependencies, common.ProfileName("full-payload-collection")) {
		t.Errorf("insights dependencies = %v, want them to include full-payload-collection", insights.Dependencies)
	}

	payload, ok := ProfilesByName["full-payload-collection"]
	if !ok {
		t.Fatal("the full-payload-collection profile is no longer declared")
	}
	if payload.MinimumTier != common.OnPremOdigosTier {
		t.Errorf("full-payload-collection MinimumTier = %q, want %q: capturing payloads must stay an enterprise feature",
			payload.MinimumTier, common.OnPremOdigosTier)
	}
	if prNameSet(GetAvailableProfilesForTier(common.CommunityOdigosTier))["full-payload-collection"] {
		t.Error("full-payload-collection is available at community tier")
	}
}
