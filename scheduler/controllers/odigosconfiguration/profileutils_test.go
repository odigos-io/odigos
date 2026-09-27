package odigosconfiguration

import (
	"slices"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/manifests"
	"github.com/odigos-io/odigos/profiles/profile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func pfSortedNames(names []common.ProfileName) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, string(n))
	}
	slices.Sort(out)
	return out
}

// TestCalculateEffectiveProfiles_InsightsPullsPayloadCollectionOnOnPrem drives the real
// profile registry through the resolver the controller uses. The insights profile has no
// manifest and no ModifyConfigFunc of its own: everything it does, it does through its
// dependencies, so the resolved list is the only place the bundle's actual contents can be
// observed. It includes full-payload-collection, which turns on capture of HTTP request and
// response bodies, database queries and messaging payloads.
func TestCalculateEffectiveProfiles_InsightsPullsPayloadCollectionOnOnPrem(t *testing.T) {
	available := profiles.GetAvailableProfilesForTier(common.OnPremOdigosTier)

	got := calculateEffectiveProfiles([]common.ProfileName{"insights"}, available)

	for _, want := range []common.ProfileName{"insights", "full-payload-collection", "infer-db-attributes", "url-template"} {
		if !slices.Contains(got, want) {
			t.Errorf("effective profiles for insights = %v, want them to include %q", pfSortedNames(got), want)
		}
	}
}

// TestCalculateEffectiveProfiles_InsightsIsUnavailableOnCommunity is the other half of the
// entitlement gate: a community cluster that lists an on-prem profile must get nothing at all
// from it, and in particular must not reach its on-prem dependencies. Without both halves a
// gate that always grants and a gate that always denies look the same.
func TestCalculateEffectiveProfiles_InsightsIsUnavailableOnCommunity(t *testing.T) {
	available := profiles.GetAvailableProfilesForTier(common.CommunityOdigosTier)

	got := calculateEffectiveProfiles([]common.ProfileName{"insights"}, available)

	if len(got) != 0 {
		t.Errorf("effective profiles for insights at community tier = %v, want none", pfSortedNames(got))
	}
}

// TestCalculateEffectiveProfiles_DoesNotResolveDependenciesAboveTheTier pins that dependency
// expansion is itself tier gated, not just the top-level profile list. Resolving dependencies
// against the unfiltered registry would let any community-available bundle hand out on-prem
// presets, so this drives a bundle that is available while one of its dependencies is not.
func TestCalculateEffectiveProfiles_DoesNotResolveDependenciesAboveTheTier(t *testing.T) {
	bundle := profile.Profile{
		ProfileName:      "test-bundle",
		MinimumTier:      common.CommunityOdigosTier,
		ShortDescription: "bundle under test",
		Dependencies:     []common.ProfileName{"test-allowed-dep", "test-gated-dep"},
	}
	allowed := profile.Profile{
		ProfileName:      "test-allowed-dep",
		MinimumTier:      common.CommunityOdigosTier,
		ShortDescription: "available at the caller's tier",
	}
	// test-gated-dep is deliberately absent from the available list, which is what the tier
	// filter does to a dependency the cluster is not entitled to.
	available := []profile.Profile{bundle, allowed}

	got := calculateEffectiveProfiles([]common.ProfileName{"test-bundle"}, available)

	want := []string{"test-allowed-dep", "test-bundle"}
	if !slices.Equal(pfSortedNames(got), want) {
		t.Errorf("effective profiles = %v, want %v", pfSortedNames(got), want)
	}
}

// TestCalculateEffectiveProfiles_ResolvesTransitiveDependencies pins that expansion is
// recursive rather than one level deep. A bundle that depends on a bundle is how the on-prem
// presets are composed, and stopping after one level would apply the intermediate bundle's own
// manifest while silently dropping everything underneath it.
func TestCalculateEffectiveProfiles_ResolvesTransitiveDependencies(t *testing.T) {
	available := []profile.Profile{
		{ProfileName: "test-top", MinimumTier: common.CommunityOdigosTier, Dependencies: []common.ProfileName{"test-middle"}},
		{ProfileName: "test-middle", MinimumTier: common.CommunityOdigosTier, Dependencies: []common.ProfileName{"test-leaf"}},
		{ProfileName: "test-leaf", MinimumTier: common.CommunityOdigosTier},
	}

	got := calculateEffectiveProfiles([]common.ProfileName{"test-top"}, available)

	want := []string{"test-leaf", "test-middle", "test-top"}
	if !slices.Equal(pfSortedNames(got), want) {
		t.Errorf("effective profiles = %v, want %v", pfSortedNames(got), want)
	}
}

// TestCalculateEffectiveProfiles_DropsUnknownNames pins that an unrecognized name is skipped
// instead of failing the reconcile. Profile names reach here from user-supplied helm values and
// from the comma separated list carried in the on-prem token, so a name this build does not
// know about must not be able to stall the configuration of the whole cluster.
func TestCalculateEffectiveProfiles_DropsUnknownNames(t *testing.T) {
	available := profiles.GetAvailableProfilesForTier(common.OnPremOdigosTier)

	got := calculateEffectiveProfiles([]common.ProfileName{"no-such-profile", "url-template", ""}, available)

	if !slices.Equal(pfSortedNames(got), []string{"url-template"}) {
		t.Errorf("effective profiles = %v, want only url-template", pfSortedNames(got))
	}

	empty := calculateEffectiveProfiles(nil, available)
	if empty == nil {
		t.Error("calculateEffectiveProfiles(nil, ...) returned a nil slice; the result is stored in the effective configuration and ranged over")
	}
	if len(empty) != 0 {
		t.Errorf("calculateEffectiveProfiles(nil, ...) = %v, want empty", pfSortedNames(empty))
	}
}

// TestEveryProfileManifestIsAppliableByTheScheduler is the cross-module contract between the
// manifests shipped by the profiles module and the kind-to-resource table this package uses to
// apply them. applyProfileManifests returns an error for an unrecognized kind, and that error
// aborts the entire reconcile before the effective configuration is written - so a manifest
// introducing a new kind does not degrade one profile, it stops every cluster that enables that
// profile from getting any configuration at all.
func TestEveryProfileManifestIsAppliableByTheScheduler(t *testing.T) {
	if len(supportedKindToResource) == 0 {
		t.Fatal("supportedKindToResource is empty")
	}

	checked := 0
	for _, p := range profiles.AllProfiles {
		yamls, err := manifests.ReadProfileYamlManifests(p.ProfileName)
		if err != nil {
			t.Errorf("reading manifests for %q: %v", p.ProfileName, err)
			continue
		}
		for _, yamlBytes := range yamls {
			obj := &unstructured.Unstructured{}
			if err := yaml.Unmarshal(yamlBytes, obj); err != nil {
				t.Errorf("profile %q manifest does not parse: %v", p.ProfileName, err)
				continue
			}
			checked++

			gvk := obj.GroupVersionKind()
			if _, ok := supportedKindToResource[gvk.Kind]; !ok {
				t.Errorf("profile %q ships a manifest of kind %q, which this package cannot map to a resource; applying it fails the whole reconcile",
					p.ProfileName, gvk.Kind)
			}
			if gvk.Group == "" || gvk.Version == "" {
				t.Errorf("profile %q manifest has no apiVersion (group %q version %q)", p.ProfileName, gvk.Group, gvk.Version)
			}
			// The name is the key the server-side apply is addressed with, so an unnamed
			// manifest would be applied under an empty name.
			if obj.GetName() == "" {
				t.Errorf("profile %q manifest has no metadata.name", p.ProfileName)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no profile manifests were checked, the test is vacuous")
	}
}

// TestCalculateProfilesDeploymentHash_IgnoresOrdering pins the property the sort in the hash
// exists for. The hash is written as a label on every resource a profile applies, and after
// applying, this package deletes every profile-managed resource whose label does not match the
// current hash. An order sensitive hash would therefore make an unchanged cluster delete and
// recreate all of its profile-managed processors, actions and instrumentation rules whenever
// the profile list happened to be resolved in a different order.
func TestCalculateProfilesDeploymentHash_IgnoresOrdering(t *testing.T) {
	const version = "v1.2.3"
	forward := calculateProfilesDeploymentHash([]common.ProfileName{"kratos", "db-payload-collection", "semconv"}, version)
	reversed := calculateProfilesDeploymentHash([]common.ProfileName{"semconv", "db-payload-collection", "kratos"}, version)

	if forward == "" {
		t.Fatal("calculateProfilesDeploymentHash returned an empty hash")
	}
	if forward != reversed {
		t.Errorf("hash depends on the order of the profile list: %q vs %q", forward, reversed)
	}
}

// TestCalculateProfilesDeploymentHash_ChangesWithTheDeployment pins the other half: the hash
// must change when the set of profiles or the odigos version changes, because that is the only
// signal that makes the previous generation of profile-managed resources get cleaned up. A
// constant hash would leave resources from removed profiles applied to the cluster forever.
func TestCalculateProfilesDeploymentHash_ChangesWithTheDeployment(t *testing.T) {
	base := []common.ProfileName{"kratos", "semconv"}
	const version = "v1.2.3"

	hash := calculateProfilesDeploymentHash(base, version)

	for _, tc := range []struct {
		name     string
		profiles []common.ProfileName
		version  string
	}{
		{name: "a profile added", profiles: []common.ProfileName{"kratos", "semconv", "copy-scope"}, version: version},
		{name: "a profile removed", profiles: []common.ProfileName{"kratos"}, version: version},
		{name: "a profile replaced", profiles: []common.ProfileName{"kratos", "semconvredis"}, version: version},
		{name: "no profiles", profiles: nil, version: version},
		{name: "odigos upgraded", profiles: base, version: "v1.2.4"},
		{name: "odigos version unset", profiles: base, version: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := calculateProfilesDeploymentHash(tc.profiles, tc.version); got == hash {
				t.Errorf("hash is unchanged (%q) after %s, so stale profile resources are never collected", got, tc.name)
			}
		})
	}
}
