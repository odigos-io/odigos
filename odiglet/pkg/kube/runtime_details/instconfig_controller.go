package runtime_details

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	var instConfig odigosv1.InstrumentationConfig
	err := i.Get(ctx, request.NamespacedName, &instConfig)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get InstrumentationConfig")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if len(instConfig.OwnerReferences) != 1 {
		return reconcile.Result{}, fmt.Errorf("InstrumentationConfig %s/%s has %d owner references, expected 1", instConfig.Namespace, instConfig.Name, len(instConfig.OwnerReferences))
	}

	workload, labels, err := getWorkloadAndLabelsfromOwner(ctx, i.Client, instConfig.Namespace, instConfig.OwnerReferences[0])
	return inspectRuntimesOfRunningPods(ctx, &logger, labels, i.Client, i.Scheme, workload)
}

func getWorkloadAndLabelsfromOwner(ctx context.Context, k8sClient client.Client, ns string, ownerReference metav1.OwnerReference) (client.Object, map[string]string, error) {
	workloadName, workloadKind, err := getWorkloadNameFromOwnerReference(ownerReference)
	if err != nil {
		return nil, nil, err
	}

	switch workloadKind {
	case "Deployment":
		var dep appsv1.Deployment
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &dep)
		if err != nil {
			return nil, nil, err
		}
		return &dep, dep.Spec.Selector.MatchLabels, nil
	case "DaemonSet":
		var ds appsv1.DaemonSet
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &ds)
		if err != nil {
			return nil, nil, err
		}

		return &ds, ds.Spec.Selector.MatchLabels, nil
	case "StatefulSet":
		var sts appsv1.StatefulSet
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &sts)
		if err != nil {
			return nil, nil, err
		}

		return &sts, sts.Spec.Selector.MatchLabels, nil
	}

	return nil, nil, errors.New("workload kind not supported")
}

func getWorkloadNameFromOwnerReference(ownerReference metav1.OwnerReference) (string, string, error) {
	name := ownerReference.Name
	kind := ownerReference.Kind
	if kind == "ReplicaSet" {
		// ReplicaSet name is in the format <deployment-name>-<random-string>
		hyphenIndex := strings.LastIndex(name, "-")
		if hyphenIndex == -1 {
			// It is possible for a user to define a bare ReplicaSet without a deployment, currently not supporting this
			return "", "", errors.New("replicaset name does not contain a hyphen")
		}
		// Extract deployment name from ReplicaSet name
		return name[:hyphenIndex], kind, nil
	} else if kind == "DaemonSet" || kind == "Deployment" || kind == "StatefulSet" {
		return name, kind, nil
	}
	return "", "", fmt.Errorf("kind %s not supported", kind)
}
