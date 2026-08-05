package recommendations

import (
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
	if len(rec.Actions) != 2 {
		t.Fatalf("Actions len = %d, want 2", len(rec.Actions))
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

	for _, rec := range Get() {
		if rec.K8sObjectName == "" {
			t.Fatalf("recommendation %q missing k8sObjectName", rec.Type)
		}
	}
}
