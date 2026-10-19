package common

import "testing"

func TestMergeInterrogationSettings(t *testing.T) {
	base := &InterrogationConfiguration{
		Enabled: boolValue(true),
		Model: &InterrogationModelConfiguration{
			Provider: stringValue("openai"), Name: stringValue("gpt-4o-mini"), ApiBase: stringValue("https://old.example/v1"),
		},
	}
	overlay := &InterrogationConfiguration{
		Enabled:              boolValue(false),
		Model:                &InterrogationModelConfiguration{ApiBase: stringValue("")},
		CredentialSecretName: "forged",
	}
	MergeInterrogationSettings(base, overlay)
	if !*base.Enabled || *base.Model.Provider != "openai" || *base.Model.Name != "gpt-4o-mini" || *base.Model.ApiBase != "" {
		t.Fatalf("field-presence merge is incorrect: %+v", base)
	}
	if base.CredentialSecretName != "" {
		t.Fatal("merge copied credential metadata")
	}
}

func boolValue(v bool) *bool       { return &v }
func stringValue(v string) *string { return &v }
