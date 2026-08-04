package recommendations

import "testing"

func TestLoad(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	recs := Get()
	if len(recs) != 5 {
		t.Fatalf("Get() len = %d, want 5", len(recs))
	}

	rec, ok := GetByType(RecommendationTypeSampleHealthProbes)
	if !ok {
		t.Fatal("GetByType(SampleHealthProbes) not found")
	}
	if len(rec.Actions) != 2 {
		t.Fatalf("Actions len = %d, want 2", len(rec.Actions))
	}

	rec, ok = GetByType(RecommendationTypeEnableOwnMetrics)
	if !ok {
		t.Fatal("GetByType(EnableOwnMetrics) not found")
	}
	if !rec.RequireOdigosDeployment {
		t.Fatal("RequireOdigosDeployment = false, want true")
	}

	rec, ok = GetByType(RecommendationTypeAutoGoOffsetUpdater)
	if !ok {
		t.Fatal("GetByType(AutoGoOffsetUpdater) not found")
	}
	if rec.OSS {
		t.Fatal("OSS = true, want false")
	}
	if len(rec.Conditions) != 1 || rec.Conditions[0].Type != "GoEnterpriseSources" {
		t.Fatalf("Conditions = %+v, want GoEnterpriseSources", rec.Conditions)
	}
}
