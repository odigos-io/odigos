package recommendations

import (
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
)

func TestLoad(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	recs := Get()
	if len(recs) != 5 {
		t.Fatalf("Get() len = %d, want 5", len(recs))
	}

	rec, ok := GetByType(common.RecommendationTypeSampleHealthProbes)
	if !ok {
		t.Fatal("GetByType(SampleHealthProbes) not found")
	}
	if len(rec.Remediations) != 2 {
		t.Fatalf("Remediations len = %d, want 2", len(rec.Remediations))
	}
	if rec.Remediations[0].Type != "DropAllHealthProbes" {
		t.Fatalf("Remediations[0].Type = %q, want DropAllHealthProbes", rec.Remediations[0].Type)
	}
	if rec.Remediations[1].Type != "SparseSampleHealthProbes" {
		t.Fatalf("Remediations[1].Type = %q, want SparseSampleHealthProbes", rec.Remediations[1].Type)
	}
	if len(rec.Remediations[0].Steps) != 2 {
		t.Fatalf("Remediations[0].Steps len = %d, want 2", len(rec.Remediations[0].Steps))
	}
	if rec.Remediations[0].Steps[0].Type != RemediationStepTypeEditConfig {
		t.Fatalf("Remediations[0].Steps[0].Type = %q, want %q", rec.Remediations[0].Steps[0].Type, RemediationStepTypeEditConfig)
	}
	if rec.Remediations[0].Steps[0].Path != "sampling.k8sHealthProbesSampling.enabled" {
		t.Fatalf("Remediations[0].Steps[0].Path = %q, want sampling.k8sHealthProbesSampling.enabled", rec.Remediations[0].Steps[0].Path)
	}
	if rec.Remediations[0].Steps[0].Value != true {
		t.Fatalf("Remediations[0].Steps[0].Value = %#v, want true", rec.Remediations[0].Steps[0].Value)
	}
	if rec.Remediations[1].Steps[1].Path != "sampling.k8sHealthProbesSampling.keepPercentage" {
		t.Fatalf("Remediations[1].Steps[1].Path = %q, want sampling.k8sHealthProbesSampling.keepPercentage", rec.Remediations[1].Steps[1].Path)
	}
	if rec.Remediations[1].Steps[1].Value != 2 {
		t.Fatalf("Remediations[1].Steps[1].Value = %#v, want 2", rec.Remediations[1].Steps[1].Value)
	}
	if len(rec.Remediations[0].ApplyExamples) != 1 {
		t.Fatalf("Remediations[0].ApplyExamples len = %d, want 1", len(rec.Remediations[0].ApplyExamples))
	}
	if rec.Remediations[0].ApplyExamples[0].Type != ApplyExampleTypeHelmValues {
		t.Fatalf("ApplyExamples[0].Type = %q, want %q", rec.Remediations[0].ApplyExamples[0].Type, ApplyExampleTypeHelmValues)
	}
	if !strings.Contains(rec.Remediations[0].ApplyExamples[0].Content, "k8sHealthProbesSampling") {
		t.Fatalf("ApplyExamples[0].Content missing k8sHealthProbesSampling: %q", rec.Remediations[0].ApplyExamples[0].Content)
	}
	if len(rec.AppliedWhen) != 1 {
		t.Fatalf("AppliedWhen len = %d, want 1", len(rec.AppliedWhen))
	}
	if rec.AppliedWhen[0].Type != AppliedWhenTypeEffectiveConfig {
		t.Fatalf("AppliedWhen[0].Type = %q, want %q", rec.AppliedWhen[0].Type, AppliedWhenTypeEffectiveConfig)
	}
	if rec.AppliedWhen[0].Expression != "sampling.k8sHealthProbesSampling.enabled == `true`" {
		t.Fatalf("AppliedWhen[0].Expression = %q, want sampling.k8sHealthProbesSampling.enabled == `true`", rec.AppliedWhen[0].Expression)
	}

	rec, ok = GetByType(common.RecommendationTypeEnableOwnMetrics)
	if !ok {
		t.Fatal("GetByType(EnableOwnMetrics) not found")
	}
	if !rec.RequireOdigosDeployment {
		t.Fatal("RequireOdigosDeployment = false, want true")
	}

	rec, ok = GetByType(common.RecommendationTypeAutoGoOffsetUpdater)
	if !ok {
		t.Fatal("GetByType(AutoGoOffsetUpdater) not found")
	}
	if rec.OSS {
		t.Fatal("OSS = true, want false")
	}
	if rec.K8sObjectName != "go-offset-updater" {
		t.Fatalf("K8sObjectName = %q, want go-offset-updater", rec.K8sObjectName)
	}
	if len(rec.Conditions) != 1 || rec.Conditions[0].Type != "GoEnterpriseSources" {
		t.Fatalf("Conditions = %+v, want GoEnterpriseSources", rec.Conditions)
	}

	rec, ok = GetByK8sObjectName("url-templatization")
	if !ok {
		t.Fatal("GetByK8sObjectName(url-templatization) not found")
	}
	if rec.Type != common.RecommendationTypeUrlTemplatization {
		t.Fatalf("Type = %q, want UrlTemplatization", rec.Type)
	}
	if len(rec.Remediations) != 1 || rec.Remediations[0].Type != "EnableUrlTemplatization" {
		t.Fatalf("Remediations = %+v, want EnableUrlTemplatization", rec.Remediations)
	}
	if len(rec.Remediations[0].ApplyExamples) != 1 || rec.Remediations[0].ApplyExamples[0].Type != ApplyExampleTypeOdigosAction {
		t.Fatalf("ApplyExamples = %+v, want OdigosAction", rec.Remediations[0].ApplyExamples)
	}

	rec, ok = GetByType(common.RecommendationTypeInferDBAttributes)
	if !ok {
		t.Fatal("GetByType(InferDBAttributes) not found")
	}
	if len(rec.Remediations) != 1 || rec.Remediations[0].ApplyExamples[0].Type != ApplyExampleTypeOdigosAction {
		t.Fatalf("InferDBAttributes ApplyExamples = %+v, want OdigosAction", rec.Remediations)
	}

	for _, rec := range Get() {
		if rec.K8sObjectName == "" {
			t.Fatalf("recommendation %q missing k8sObjectName", rec.Type)
		}
	}
}
