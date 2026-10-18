package common

import "testing"

func TestMergeInterrogationSettingsAndModelHash(t *testing.T) {
	base := &InterrogationConfiguration{
		Enabled: boolValue(true),
		Model: &InterrogationModelConfiguration{
			Provider: stringValue("openai"), Name: stringValue("gpt-4o-mini"), ApiBase: stringValue("https://old.example/v1"),
		},
	}
	overlay := &InterrogationConfiguration{
		Enabled: boolValue(false), AnalysisEnabled: boolValue(false),
		Model:                &InterrogationModelConfiguration{ApiBase: stringValue("")},
		CredentialSecretName: "forged", UpdateID: "forged",
	}
	MergeInterrogationSettings(base, overlay)
	if !*base.Enabled || *base.AnalysisEnabled || *base.Model.Provider != "openai" || *base.Model.Name != "gpt-4o-mini" || *base.Model.ApiBase != "" {
		t.Fatalf("field-presence merge is incorrect: %+v", base)
	}
	if base.CredentialSecretName != "" || base.UpdateID != "" {
		t.Fatal("merge copied credential metadata")
	}
	if got, want := InterrogationModelHash(base.Model), InterrogationModelHash(nil); got != want {
		t.Errorf("explicit defaults and nil defaults have different hashes: %s != %s", got, want)
	}
	base.Model.Name = stringValue(" gpt-4.1 ")
	if InterrogationModelHash(base.Model) == InterrogationModelHash(nil) {
		t.Fatal("model change did not change hash")
	}
	trimmed := &InterrogationModelConfiguration{Name: stringValue("gpt-4.1")}
	if InterrogationModelHash(base.Model) != InterrogationModelHash(trimmed) {
		t.Fatal("model hash did not normalize surrounding whitespace")
	}
}

func boolValue(v bool) *bool       { return &v }
func stringValue(v string) *string { return &v }
