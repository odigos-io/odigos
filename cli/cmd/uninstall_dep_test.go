package cmd

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func source(namespace string, name string) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
}

// listSourcesFailsWith makes every `list sources` call answer with err. The generated client
// still hands back a freshly allocated, empty SourceList alongside the error (see
// client-go gentype: the list object is built before the request is issued), which is exactly
// the shape removeAllSources has to reason about.
func listSourcesFailsWith(t *testing.T, err error, objects ...runtime.Object) *kube.Client {
	t.Helper()
	clientset := odigosfake.NewSimpleClientset(objects...)
	clientset.PrependReactor("list", "sources", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, err
	})
	return &kube.Client{OdigosClient: clientset.OdigosV1alpha1()}
}

var sourcesResource = odigosv1.Resource("sources")

// A failed list must not be reported as "no sources to delete". The uninstall flow goes on to
// delete the Source CRD and the instrumentor once this returns nil, which would leave every
// workload instrumented and the CRD wedged behind its finalizers.
func TestRemoveAllSourcesPropagatesListFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "server error", err: apierrors.NewInternalError(context.DeadlineExceeded)},
		{name: "forbidden", err: apierrors.NewForbidden(sourcesResource, "", context.DeadlineExceeded)},
		{name: "unavailable", err: apierrors.NewServiceUnavailable("etcd leader changed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := listSourcesFailsWith(t, tc.err, source("app", "deployment-frontend"))

			if err := removeAllSources(context.Background(), client); err == nil {
				t.Fatal("expected the list failure to be returned, got nil (uninstall would continue and report success)")
			}

			remaining, err := client.OdigosClient.Sources("").List(context.Background(), metav1.ListOptions{})
			if err == nil && len(remaining.Items) != 1 {
				t.Fatalf("expected the source to still exist, got %d", len(remaining.Items))
			}
		})
	}
}

// The one error that genuinely means "nothing to delete": the CRD itself is already gone.
// This is what the poll loop further down the same function already does.
func TestRemoveAllSourcesToleratesMissingCRD(t *testing.T) {
	client := listSourcesFailsWith(t, apierrors.NewNotFound(sourcesResource, ""))

	if err := removeAllSources(context.Background(), client); err != nil {
		t.Fatalf("a missing Source CRD means there is nothing to delete, got %v", err)
	}
}

func TestRemoveAllSourcesDeletesEverySource(t *testing.T) {
	clientset := odigosfake.NewSimpleClientset(
		source("app", "deployment-frontend"),
		source("payments", "deployment-api"),
		source("payments", "daemonset-agent"),
	)
	client := &kube.Client{OdigosClient: clientset.OdigosV1alpha1()}

	if err := removeAllSources(context.Background(), client); err != nil {
		t.Fatalf("removeAllSources: %v", err)
	}

	remaining, err := client.OdigosClient.Sources("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(remaining.Items) != 0 {
		t.Fatalf("expected every source to be deleted, %d left", len(remaining.Items))
	}
}
