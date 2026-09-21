package cmd

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func testSource(name string, disableInstrumentation bool) *v1alpha1.Source {
	return &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      name,
				k8sconsts.WorkloadNamespaceLabel: "default",
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindDeployment),
			},
		},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Kind:      k8sconsts.WorkloadKindDeployment,
				Name:      name,
				Namespace: "default",
			},
			DisableInstrumentation: disableInstrumentation,
		},
	}
}

// runSourceUpdate parses args onto the update command and runs it against a fake Odigos client,
// returning the Sources as they were left in the cluster.
func runSourceUpdate(t *testing.T, args []string, sources ...*v1alpha1.Source) map[string]v1alpha1.Source {
	t.Helper()

	objects := make([]runtime.Object, 0, len(sources))
	for _, source := range sources {
		objects = append(objects, source)
	}
	fakeClient := fake.NewSimpleClientset(objects...)

	sourceUpdateCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err := f.Value.Set(f.DefValue); err != nil {
			t.Fatalf("cannot reset flag %s: %v", f.Name, err)
		}
		f.Changed = false
	})
	if err := sourceUpdateCmd.ParseFlags(args); err != nil {
		t.Fatalf("cannot parse flags: %v", err)
	}

	ctx := cmdcontext.ContextWithKubeClient(context.Background(), &kube.Client{
		OdigosClient: fakeClient.OdigosV1alpha1(),
	})
	sourceUpdateCmd.SetContext(ctx)
	sourceUpdateCmd.Run(sourceUpdateCmd, nil)

	updated, err := fakeClient.OdigosV1alpha1().Sources("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("cannot list Sources: %v", err)
	}
	result := make(map[string]v1alpha1.Source, len(updated.Items))
	for _, source := range updated.Items {
		result[source.GetName()] = source
	}
	return result
}

func TestSourceUpdateKeepsDisableInstrumentationWhenFlagIsNotProvided(t *testing.T) {
	sources := runSourceUpdate(t,
		[]string{"--set-group", "mygroup", "--yes"},
		testSource("disabled-source", true),
		testSource("enabled-source", false),
	)

	if !sources["disabled-source"].Spec.DisableInstrumentation {
		t.Error("updating an unrelated field re-enabled instrumentation on a disabled Source")
	}
	if sources["enabled-source"].Spec.DisableInstrumentation {
		t.Error("updating an unrelated field disabled instrumentation on an enabled Source")
	}
	for name, source := range sources {
		if source.Labels[k8sconsts.SourceDataStreamLabelPrefix+"mygroup"] != "true" {
			t.Errorf("Source %s was not added to the group", name)
		}
	}
}

func TestSourceUpdateAppliesDisableInstrumentationWhenFlagIsProvided(t *testing.T) {
	sources := runSourceUpdate(t,
		[]string{"--disable-instrumentation", "--yes"},
		testSource("enabled-source", false),
	)
	if !sources["enabled-source"].Spec.DisableInstrumentation {
		t.Error("--disable-instrumentation did not disable instrumentation")
	}

	sources = runSourceUpdate(t,
		[]string{"--disable-instrumentation=false", "--yes"},
		testSource("disabled-source", true),
	)
	if sources["disabled-source"].Spec.DisableInstrumentation {
		t.Error("--disable-instrumentation=false did not enable instrumentation")
	}
}
