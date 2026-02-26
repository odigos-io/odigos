package inspectors

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type fakeInspector struct {
	lang          common.ProgrammingLanguage
	quickDetected bool
	deepDetected  bool
}

func (f *fakeInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return f.lang, f.quickDetected
}

func (f *fakeInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return f.lang, f.deepDetected
}

type fakeVersionInspector struct {
	lang          common.ProgrammingLanguage
	quickDetected bool
	deepDetected  bool
	versionString string
}

func (f *fakeVersionInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return f.lang, f.quickDetected
}

func (f *fakeVersionInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return f.lang, f.deepDetected
}

func (f *fakeVersionInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	return f.versionString
}

func TestConflictWithCppPrefersOtherLanguage_QuickScan(t *testing.T) {
	orig := inspectorsByLanguage
	defer func() { inspectorsByLanguage = orig }()

	inspectorsByLanguage = map[common.ProgrammingLanguage]Inspector{
		common.GoProgrammingLanguage:        &fakeVersionInspector{lang: common.GoProgrammingLanguage, quickDetected: true, deepDetected: false, versionString: "1.20.0"},
		common.CPlusPlusProgrammingLanguage: &fakeInspector{lang: common.CPlusPlusProgrammingLanguage, quickDetected: true, deepDetected: false},
	}

	res, err := DetectLanguage(process.Details{}, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Language != common.GoProgrammingLanguage {
		t.Fatalf("expected language %s, got %s", common.GoProgrammingLanguage, res.Language)
	}
	if res.RuntimeVersion != "1.20.0" {
		t.Fatalf("expected runtime version 1.20.0, got %v", res.RuntimeVersion)
	}
}

func TestConflictWithoutCppReturnsError_QuickScan(t *testing.T) {
	orig := inspectorsByLanguage
	defer func() { inspectorsByLanguage = orig }()

	inspectorsByLanguage = map[common.ProgrammingLanguage]Inspector{
		common.GoProgrammingLanguage:     &fakeInspector{lang: common.GoProgrammingLanguage, quickDetected: true, deepDetected: false},
		common.PythonProgrammingLanguage: &fakeInspector{lang: common.PythonProgrammingLanguage, quickDetected: true, deepDetected: false},
	}

	res, err := DetectLanguage(process.Details{}, "")
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
	if _, ok := err.(ErrLanguageDetectionConflict); !ok {
		t.Fatalf("expected ErrLanguageDetectionConflict, got %T", err)
	}
	if res.Language != common.UnknownProgrammingLanguage {
		t.Fatalf("expected unknown language, got %s", res.Language)
	}
}

func TestConflictWithCppPrefersOtherLanguage_DeepScan(t *testing.T) {
	orig := inspectorsByLanguage
	defer func() { inspectorsByLanguage = orig }()

	inspectorsByLanguage = map[common.ProgrammingLanguage]Inspector{
		common.GoProgrammingLanguage:        &fakeVersionInspector{lang: common.GoProgrammingLanguage, quickDetected: false, deepDetected: true, versionString: "1.19.5"},
		common.CPlusPlusProgrammingLanguage: &fakeInspector{lang: common.CPlusPlusProgrammingLanguage, quickDetected: false, deepDetected: true},
	}

	res, err := DetectLanguage(process.Details{}, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Language != common.GoProgrammingLanguage {
		t.Fatalf("expected language %s, got %s", common.GoProgrammingLanguage, res.Language)
	}
	if res.RuntimeVersion != "1.19.5" {
		t.Fatalf("expected runtime version 1.19.5, got %v", res.RuntimeVersion)
	}
}
