package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
)

func TestCalculatePayloadCollectionConfig_IgnoresMismatchedLibraryLanguage(t *testing.T) {
	mime := []string{"application/json"}
	maxLen := int64(1024)

	irls := []odigosv1.InstrumentationRule{
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				InstrumentationLibraries: &[]odigosv1.InstrumentationLibraryGlobalId{
					{
						Name:     "requests",
						Language: common.PythonProgrammingLanguage,
					},
				},
				PayloadCollection: &instrumentationrules.PayloadCollection{
					HttpRequest: &instrumentationrules.HttpPayloadCollection{
						MimeTypes:        &mime,
						MaxPayloadLength: &maxLen,
					},
				},
			},
		},
	}

	d := &distro.OtelDistro{
		Traces: &distro.Traces{
			PayloadCollection: &distro.PayloadCollection{
				Supported: true,
			},
		},
	}

	got := CalculatePayloadCollectionConfig(d, &irls, common.JavascriptProgrammingLanguage)
	if got != nil {
		t.Fatalf("expected nil payload config for non-matching language, got: %+v", got)
	}
}

func TestCalculatePayloadCollectionConfig_DoesNotMutateSourceRule(t *testing.T) {
	mime := []string{"application/json"}
	maxLen := int64(1024)
	drop := true
	secondaryMaxLen := int64(256)

	irls := []odigosv1.InstrumentationRule{
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				PayloadCollection: &instrumentationrules.PayloadCollection{
					HttpRequest: &instrumentationrules.HttpPayloadCollection{
						MimeTypes:           &mime,
						MaxPayloadLength:    &maxLen,
						DropPartialPayloads: &drop,
					},
				},
			},
		},
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				PayloadCollection: &instrumentationrules.PayloadCollection{
					HttpRequest: &instrumentationrules.HttpPayloadCollection{
						MaxPayloadLength: &secondaryMaxLen,
					},
				},
			},
		},
	}

	original := irls[0].Spec.PayloadCollection.DeepCopy()

	d := &distro.OtelDistro{
		Traces: &distro.Traces{
			PayloadCollection: &distro.PayloadCollection{
				Supported: true,
			},
		},
	}

	_ = CalculatePayloadCollectionConfig(d, &irls, common.JavascriptProgrammingLanguage)

	got := irls[0].Spec.PayloadCollection
	if got == nil || got.HttpRequest == nil || got.HttpRequest.MaxPayloadLength == nil {
		t.Fatalf("unexpected nil source rule after calculation: %+v", got)
	}
	if *got.HttpRequest.MaxPayloadLength != *original.HttpRequest.MaxPayloadLength {
		t.Fatalf("source rule was mutated: max payload length changed from %d to %d", *original.HttpRequest.MaxPayloadLength, *got.HttpRequest.MaxPayloadLength)
	}
}
