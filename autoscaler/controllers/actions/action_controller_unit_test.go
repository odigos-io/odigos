package actions

import (
	"context"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileDeleteSyncsSharedURLTemplatizationProcessor(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, "default")

	scheme := runtime.NewScheme()
	if err := odigosv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add odigos scheme: %v", err)
	}

	sharedProcessor := &odigosv1.Processor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.URLTemplatizationProcessorName,
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sharedProcessor).
		Build()

	r := &ActionReconciler{Client: c}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "deleted-action",
			Namespace: "default",
		},
	})
	if err != nil {
		t.Fatalf("reconcile delete request: %v", err)
	}

	err = c.Get(context.Background(), types.NamespacedName{
		Name:      consts.URLTemplatizationProcessorName,
		Namespace: "default",
	}, &odigosv1.Processor{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected shared URL templatization processor to be deleted, got err=%v", err)
	}
}
