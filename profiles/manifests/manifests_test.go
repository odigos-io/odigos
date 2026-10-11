package manifests

import (
	"bytes"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
)

// TestReadProfileYamlManifestsReturnsNoElementsWhenThereIsNoManifest is the contract the
// scheduler depends on for every profile that has no yaml of its own - the aggregators
// (kratos, greatwall, insights) and the profiles that work purely through a ModifyConfigFunc.
// The scheduler ranges over the returned slice and unmarshals each element into an
// unstructured object, and an empty document unmarshals without error into an object with no
// kind, which the scheduler then rejects with "unsupported kind for profile manifest".
//
// So returning a single empty element instead of no elements would not degrade one profile: it
// would abort the whole odigos configuration reconcile for every cluster that enables an
// aggregator profile, which is every on-prem cluster.
func TestReadProfileYamlManifestsReturnsNoElementsWhenThereIsNoManifest(t *testing.T) {
	for _, name := range []common.ProfileName{
		"kratos",
		"greatwall",
		"insights",
		"allow_concurrent_agents",
		"mount-method-k8s-host-path",
		"no-such-profile",
		// A name that looks like a manifest that does exist, to pin that the lookup is exact.
		"full-payload-collection.bak",
		"Full-Payload-Collection",
	} {
		t.Run(string(name), func(t *testing.T) {
			yamls, err := ReadProfileYamlManifests(name)
			if err != nil {
				t.Fatalf("ReadProfileYamlManifests(%q) returned error %v, want nil: a profile without a manifest is not a failure", name, err)
			}
			if len(yamls) != 0 {
				t.Fatalf("ReadProfileYamlManifests(%q) returned %d elements, want 0; the scheduler rejects an empty manifest with an unsupported-kind error",
					name, len(yamls))
			}
		})
	}
}

// TestReadProfileYamlManifestsReturnsTheEmbeddedFile drives the name-to-file lookup over every
// embedded manifest, so a manifest that stops being reachable under its own basename - because
// the filename pattern changed, or because the embed directive stopped matching it - fails
// here instead of silently no longer being applied to any cluster.
func TestReadProfileYamlManifestsReturnsTheEmbeddedFile(t *testing.T) {
	entries, err := embeddedFiles.ReadDir(".")
	if err != nil {
		t.Fatalf("listing embedded manifests: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no manifests are embedded")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			t.Errorf("unexpected embedded directory %q", entry.Name())
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			t.Errorf("embedded file %q is not a .yaml file, so no profile name can ever resolve to it", entry.Name())
			continue
		}

		profileName := common.ProfileName(strings.TrimSuffix(entry.Name(), ".yaml"))
		t.Run(string(profileName), func(t *testing.T) {
			want, err := embeddedFiles.ReadFile(entry.Name())
			if err != nil {
				t.Fatalf("reading %q: %v", entry.Name(), err)
			}
			if len(want) == 0 {
				t.Fatalf("embedded manifest %q is empty", entry.Name())
			}

			got, err := ReadProfileYamlManifests(profileName)
			if err != nil {
				t.Fatalf("ReadProfileYamlManifests(%q) returned error %v", profileName, err)
			}
			if len(got) != 1 {
				t.Fatalf("ReadProfileYamlManifests(%q) returned %d elements, want exactly 1", profileName, len(got))
			}
			if !bytes.Equal(got[0], want) {
				t.Errorf("ReadProfileYamlManifests(%q) did not return the contents of %q", profileName, entry.Name())
			}
		})
	}
}
