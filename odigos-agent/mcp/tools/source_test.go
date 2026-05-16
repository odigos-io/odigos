package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestApprovalCachePutTakeRoundtrip(t *testing.T) {
	cache := NewApprovalCache(time.Minute)
	id := cache.Put(&PendingMutation{
		Operation:    "create_source",
		Namespace:    "default",
		WorkloadKind: "Deployment",
		WorkloadName: "payments",
	})
	if id == "" {
		t.Fatal("expected non-empty request_id")
	}
	if cache.Size() != 1 {
		t.Fatalf("expected size 1, got %d", cache.Size())
	}
	taken := cache.Take(id)
	if taken == nil {
		t.Fatal("Take returned nil for fresh entry")
	}
	if taken.WorkloadName != "payments" {
		t.Errorf("unexpected workload name: %q", taken.WorkloadName)
	}
	if cache.Take(id) != nil {
		t.Error("Take should be one-shot")
	}
}

func TestApprovalCacheTTLEviction(t *testing.T) {
	cache := NewApprovalCache(50 * time.Millisecond)
	clock := time.Unix(0, 0)
	cache.now = func() time.Time { return clock }

	id := cache.Put(&PendingMutation{Operation: "create_source"})
	clock = clock.Add(time.Second)
	if got := cache.Take(id); got != nil {
		t.Error("expected expired entry to be dropped")
	}
}

func TestApprovalCacheUnknownIDReturnsNil(t *testing.T) {
	cache := NewApprovalCache(time.Minute)
	if got := cache.Take("not-a-real-id"); got != nil {
		t.Error("expected nil for unknown request_id")
	}
}

func TestTailSlice(t *testing.T) {
	cases := []struct {
		name string
		in   []int
		n    int
		want []int
	}{
		{"n<=0 returns input", []int{1, 2, 3}, 0, []int{1, 2, 3}},
		{"n>len returns input", []int{1, 2}, 5, []int{1, 2}},
		{"n<len tails", []int{1, 2, 3, 4, 5}, 2, []int{4, 5}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := TailSlice(testCase.in, testCase.n)
			if len(got) != len(testCase.want) {
				t.Fatalf("len: got %d want %d", len(got), len(testCase.want))
			}
			for index := range got {
				if got[index] != testCase.want[index] {
					t.Fatalf("idx %d: got %d want %d", index, got[index], testCase.want[index])
				}
			}
		})
	}
}

func TestClampInt(t *testing.T) {
	if got := ClampInt(5, 10, 20); got != 10 {
		t.Errorf("under-low: got %d", got)
	}
	if got := ClampInt(25, 10, 20); got != 20 {
		t.Errorf("over-high: got %d", got)
	}
	if got := ClampInt(15, 10, 20); got != 15 {
		t.Errorf("in-range: got %d", got)
	}
}

func TestIsSupportedWorkloadKind(t *testing.T) {
	supported := []string{"Deployment", "StatefulSet", "DaemonSet", "CronJob", "Job", "Namespace", "DeploymentConfig", "Rollout"}
	for _, kind := range supported {
		if !IsSupportedWorkloadKind(kind) {
			t.Errorf("expected %s to be supported", kind)
		}
	}
	if IsSupportedWorkloadKind("Pod") {
		t.Error("Pod must not be supported - we never create Sources for bare Pods")
	}
	if IsSupportedWorkloadKind("") {
		t.Error("empty kind must not be supported")
	}
}

func TestPrefixLines(t *testing.T) {
	got := prefixLines("a\nb\n", "+ ")
	want := "+ a\n+ b\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if prefixLines("", "+ ") != "" {
		t.Error("empty input must produce empty output")
	}
}

func TestDescribeEnvMasksSecrets(t *testing.T) {
	env := []corev1.EnvVar{
		{Name: "PLAIN", Value: "literal-value"},
		{Name: "FROM_SECRET", ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
				Key:                  "api-key",
			},
		}},
		{Name: "FROM_CM", ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "settings"},
				Key:                  "verbosity",
			},
		}},
	}
	entries, more := describeEnv(env)
	if more {
		t.Error("expected more=false")
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[1]["source"] != "secret/creds/api-key" {
		t.Errorf("secret source mislabeled: %v", entries[1]["source"])
	}
	if _, hasValue := entries[1]["value"]; hasValue {
		t.Error("secret entry must not include a value field")
	}
	if entries[2]["source"] != "configmap/settings/verbosity" {
		t.Errorf("configmap source mislabeled: %v", entries[2]["source"])
	}
}

func TestDescribeEnvCapsAtLimit(t *testing.T) {
	env := make([]corev1.EnvVar, maxContainerEnvItems+5)
	for index := range env {
		env[index] = corev1.EnvVar{Name: "VAR"}
	}
	entries, more := describeEnv(env)
	if !more {
		t.Error("expected more=true when over the cap")
	}
	if len(entries) != maxContainerEnvItems {
		t.Errorf("expected %d entries, got %d", maxContainerEnvItems, len(entries))
	}
}

func TestFindWorkloadSourceMatchByWorkloadSpec(t *testing.T) {
	source := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: "src-abc", Namespace: "default"},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: "default",
				Name:      "payments",
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
		},
	}
	manager := &sourceManager{clients: &Clients{
		Core:   kubefake.NewSimpleClientset(),
		Odigos: odigosfake.NewSimpleClientset(source),
	}}
	got, err := manager.findWorkloadSource(context.Background(), "default", "Deployment", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Name != "src-abc" {
		t.Fatalf("expected to find src-abc, got %+v", got)
	}
}

func TestFindWorkloadSourceReturnsNilOnNoMatch(t *testing.T) {
	source := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: "src-other", Namespace: "default"},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: "default",
				Name:      "other-app",
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
		},
	}
	manager := &sourceManager{clients: &Clients{
		Core:   kubefake.NewSimpleClientset(),
		Odigos: odigosfake.NewSimpleClientset(source),
	}}
	got, err := manager.findWorkloadSource(context.Background(), "default", "Deployment", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestFindNamespaceSource(t *testing.T) {
	source := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: "ns-src", Namespace: "default"},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: "default",
				Name:      "default",
				Kind:      k8sconsts.WorkloadKindNamespace,
			},
		},
	}
	manager := &sourceManager{clients: &Clients{
		Core:   kubefake.NewSimpleClientset(),
		Odigos: odigosfake.NewSimpleClientset(source),
	}}
	got, err := manager.findNamespaceSource(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Name != "ns-src" {
		t.Fatalf("expected ns-src, got %+v", got)
	}
}

func TestBuildSourceForWorkloadShape(t *testing.T) {
	source := buildSourceForWorkload("default", "Deployment", "payments")
	if source.GenerateName != "source-" {
		t.Errorf("GenerateName: got %q want %q", source.GenerateName, "source-")
	}
	if source.Namespace != "default" {
		t.Errorf("Namespace: got %q", source.Namespace)
	}
	if source.Spec.Workload.Name != "payments" {
		t.Errorf("Workload.Name: got %q", source.Spec.Workload.Name)
	}
	if source.Spec.Workload.Kind != k8sconsts.WorkloadKindDeployment {
		t.Errorf("Workload.Kind: got %q", source.Spec.Workload.Kind)
	}
}

func TestStripServerFieldsErasesMetadata(t *testing.T) {
	original := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "x",
			Namespace:       "default",
			ResourceVersion: "12345",
			UID:             "abc-def",
			Generation:      7,
			ManagedFields:   []metav1.ManagedFieldsEntry{{Manager: "kubectl"}},
		},
	}
	stripped := stripServerFields(original)
	if stripped.ResourceVersion != "" {
		t.Error("resource version must be cleared")
	}
	if stripped.UID != "" {
		t.Error("uid must be cleared")
	}
	if stripped.Generation != 0 {
		t.Error("generation must be cleared")
	}
	if stripped.ManagedFields != nil {
		t.Error("managed fields must be cleared")
	}
	if original.ResourceVersion != "12345" {
		t.Error("stripServerFields must not mutate the input")
	}
}

func TestPrefixLinesCollapsesTrailingBlankLines(t *testing.T) {
	got := prefixLines("alpha\nbeta\n\n", "> ")
	want := "> alpha\n> beta\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("output must end with a single newline")
	}
}
