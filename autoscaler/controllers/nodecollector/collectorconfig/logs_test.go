package collectorconfig

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetReceivers_OnlyFilelogWhenNoEbpfOptIn(t *testing.T) {
	sources := &v1alpha1.InstrumentationConfigList{
		Items: []v1alpha1.InstrumentationConfig{
			newICForLogs("default", "workload-a", false),
			newICForLogs("default", "workload-b", false),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")
	if len(pipelineReceivers) != 1 || pipelineReceivers[0] != filelogReceiverName {
		t.Fatalf("expected only %q receiver, got %v", filelogReceiverName, pipelineReceivers)
	}
	if _, ok := receivers[filelogReceiverName]; !ok {
		t.Fatalf("expected %q receiver config to exist", filelogReceiverName)
	}
}

func TestGetReceivers_OnlyEbpfWhenAllWorkloadsOptIn(t *testing.T) {
	sources := &v1alpha1.InstrumentationConfigList{
		Items: []v1alpha1.InstrumentationConfig{
			newICForLogs("default", "workload-a", true),
			newICForLogs("default", "workload-b", true),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")
	if len(pipelineReceivers) != 1 || pipelineReceivers[0] != odigosEbpfReceiverName {
		t.Fatalf("expected only %q receiver, got %v", odigosEbpfReceiverName, pipelineReceivers)
	}
	if len(receivers) != 0 {
		t.Fatalf("expected no extra receiver configs, got %v", receivers)
	}
}

func TestGetReceivers_MixedOptInUsesBothAndScopesFilelogIncludes(t *testing.T) {
	sources := &v1alpha1.InstrumentationConfigList{
		Items: []v1alpha1.InstrumentationConfig{
			newICForLogs("default", "workload-ebpf", true),
			newICForLogs("other", "workload-filelog", false),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")
	if len(pipelineReceivers) != 2 ||
		pipelineReceivers[0] != odigosEbpfReceiverName ||
		pipelineReceivers[1] != filelogReceiverName {
		t.Fatalf("expected [%q %q], got %v", odigosEbpfReceiverName, filelogReceiverName, pipelineReceivers)
	}

	filelogCfgAny, ok := receivers[filelogReceiverName]
	if !ok {
		t.Fatalf("expected %q receiver config to exist", filelogReceiverName)
	}
	filelogCfg, ok := filelogCfgAny.(config.GenericMap)
	if !ok {
		t.Fatalf("expected filelog config type %T, got %T", config.GenericMap{}, filelogCfgAny)
	}

	includesAny, ok := filelogCfg["include"]
	if !ok {
		t.Fatalf("expected filelog include list to exist")
	}
	includes, ok := includesAny.([]string)
	if !ok {
		t.Fatalf("expected include list type []string, got %T", includesAny)
	}
	if len(includes) != 1 {
		t.Fatalf("expected one include pattern for non-eBPF workload, got %v", includes)
	}

	want := "/var/log/pods/other_workload-filelog-*_*/*/*.log"
	if includes[0] != want {
		t.Fatalf("unexpected include pattern, got %q want %q", includes[0], want)
	}
}

func newICForLogs(namespace string, workloadName string, ebpfEnabled bool) v1alpha1.InstrumentationConfig {
	ic := v1alpha1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment",
				Name: workloadName,
				UID:  types.UID(workloadName + "-uid"),
			}},
		},
	}

	if ebpfEnabled {
		enabled := true
		ic.Spec.SdkConfigs = []v1alpha1.SdkConfig{{
			Language: common.GoProgrammingLanguage,
			EbpfLogCapture: &instrumentationrules.EbpfLogCapture{
				Enabled: &enabled,
			},
		}}
	}

	return ic
}
