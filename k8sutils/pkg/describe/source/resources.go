package source

import (
	"context"
	"fmt"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type OdigosSourceResources struct {
	Namespace                *corev1.Namespace
	InstrumentationConfig    *odigosv1.InstrumentationConfig
	InstrumentationInstances *odigosv1.InstrumentationInstanceList
	Pods                     *corev1.PodList
}

func GetRelevantSourceResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, workloadObj *K8sSourceObject) (*OdigosSourceResources, error) {

	sourceResources := OdigosSourceResources{}

	workloadNs := workloadObj.GetNamespace()
	ns, err := kubeClient.CoreV1().Namespaces().Get(ctx, workloadNs, metav1.GetOptions{})
	if err == nil {
		sourceResources.Namespace = ns
	} else {
		// namespace must be found
		return nil, err
	}

	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName(workloadObj.GetName(), workloadObj.Kind)
	ic, err := odigosClient.InstrumentationConfigs(workloadNs).Get(ctx, runtimeObjectName, metav1.GetOptions{})
	if err == nil {
		sourceResources.InstrumentationConfig = ic
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	instrumentedAppSelector := labels.SelectorFromSet(labels.Set{
		"instrumented-app": runtimeObjectName,
	})
	iis, err := odigosClient.InstrumentationInstances(workloadNs).List(ctx, metav1.ListOptions{LabelSelector: instrumentedAppSelector.String()})
	if err == nil {
		sourceResources.InstrumentationInstances = iis
	} else {
		return nil, err
	}

	sourceResources.Pods, err = getSourcePods(ctx, kubeClient, workloadObj)
	if err != nil {
		return nil, err
	}

	return &sourceResources, nil
}

func getSourcePods(ctx context.Context, kubeClient kubernetes.Interface, workloadObj *K8sSourceObject) (*corev1.PodList, error) {
	podLabelSelector := metav1.FormatLabelSelector(workloadObj.LabelSelector)

	if workloadObj.Kind == "deployment" {
		// In case 2 deployment have the same podLabelselector and namespace, we need to get the specific pods
		// for the deployment, get the pods by listing the replica-sets owned by the deployment and then listing the pods
		replicaSets, err := kubeClient.AppsV1().ReplicaSets(workloadObj.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: podLabelSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing replicasets: %v", err)
		}

		pods := &corev1.PodList{}

		for _, rs := range replicaSets.Items {
			// Check if this ReplicaSet is owned by the deployment
			for _, ownerRef := range rs.OwnerReferences {
				if string(ownerRef.UID) == string(workloadObj.UID) && ownerRef.Kind == "Deployment" {

					// List pods for this specific ReplicaSet
					podList, err := kubeClient.CoreV1().Pods(workloadObj.Namespace).List(ctx, metav1.ListOptions{
						LabelSelector: metav1.FormatLabelSelector(rs.Spec.Selector),
					})
					if err != nil {
						return nil, fmt.Errorf("error listing pods for replicaset: %v", err)
					}

					// Add these pods to our specific pods list
					pods.Items = append(pods.Items, podList.Items...)
					break
				}
			}
		}
		return pods, nil
	} else {
		pods, err := kubeClient.CoreV1().Pods(workloadObj.Namespace).List(ctx, metav1.ListOptions{LabelSelector: podLabelSelector})
		if err != nil {
			return nil, err
		}
		return pods, nil
	}
}
