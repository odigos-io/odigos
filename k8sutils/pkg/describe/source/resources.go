package source

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

const (
	// this annotation is not really set on the pod object itself.
	// it is used as a patch to signal if the pod is running the latest revision of the deployment.
	// this is useful so we know to ignore pods that are from previous revisions which are in the process of being terminated
	// during a rolling update.
	OdigosRunningLatestWorkloadRevisionAnnotation = "odigos.io/running-latest-workload-revision"
)

type OdigosSourceResources struct {
	Namespace                *corev1.Namespace
	Sources                  *odigosv1.WorkloadSources
	InstrumentationConfig    *odigosv1.InstrumentationConfig
	InstrumentationInstances *odigosv1.InstrumentationInstanceList
	Pods                     *corev1.PodList
}

func GetRelevantSourceResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface,
	workloadObj *K8sSourceObject,
) (*OdigosSourceResources, error) {
	sourceResources := OdigosSourceResources{}

	sources, err := getSources(ctx, odigosClient, workloadObj)
	if err != nil {
		return nil, err
	}
	sourceResources.Sources = sources

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
	var podLabelSelector string
	if workloadObj.LabelSelector != nil {
		podLabelSelector = metav1.FormatLabelSelector(workloadObj.LabelSelector)
	}

	if workloadObj.Kind == k8sconsts.WorkloadKindDeployment {
		// In case 2 deployment have the same podLabelselector and namespace, we need to get the specific pods
		// for the deployment, get the pods by listing the replica-sets owned by the deployment and then listing the pods
		replicaSets, err := kubeClient.AppsV1().ReplicaSets(workloadObj.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: podLabelSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing replicasets: %v", err)
		}

		deploymentRevision := workloadObj.Annotations["deployment.kubernetes.io/revision"]

		pods := &corev1.PodList{}

		for i := range replicaSets.Items {
			rs := &replicaSets.Items[i]
			// Check if this ReplicaSet is owned by the deployment
			for _, ownerRef := range rs.OwnerReferences {
				if ownerRef.Kind != "Deployment" {
					continue
				}
				if string(ownerRef.UID) != string(workloadObj.UID) {
					continue
				}

				rsRevision := rs.Annotations["deployment.kubernetes.io/revision"]
				activeReplicaSet := deploymentRevision == rsRevision && deploymentRevision != ""
				// List pods for this specific ReplicaSet
				podList, err := kubeClient.CoreV1().Pods(workloadObj.Namespace).List(ctx, metav1.ListOptions{
					LabelSelector: metav1.FormatLabelSelector(rs.Spec.Selector),
				})
				if err != nil {
					return nil, fmt.Errorf("error listing pods for replicaset: %v", err)
				}

				isLatestRevisionText := "false"
				if activeReplicaSet {
					isLatestRevisionText = "true"
				}
				for i := range podList.Items {
					if podList.Items[i].Annotations == nil {
						podList.Items[i].Annotations = make(map[string]string)
					}
					podList.Items[i].Annotations[OdigosRunningLatestWorkloadRevisionAnnotation] = isLatestRevisionText
				}

				// Add these pods to our specific pods list
				pods.Items = append(pods.Items, podList.Items...)
				break
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

// this function is based on the GetSources function in api/odigos/v1alpha1/source_types.go
// the reason for this duplication is the different clients used.
func getSources(ctx context.Context, sourcesClient odigosclientset.SourcesGetter, obj *K8sSourceObject) (*odigosv1.WorkloadSources, error) {
	if obj == nil {
		return nil, fmt.Errorf("workload object is nil")
	}

	var err error
	workloadSources := &odigosv1.WorkloadSources{}

	namespace := obj.GetNamespace()
	if namespace == "" && obj.Kind == k8sconsts.WorkloadKindNamespace {
		namespace = obj.GetName()
	}

	if obj.Kind != k8sconsts.WorkloadKindNamespace {
		selector := labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadNameLabel:      obj.GetName(),
			k8sconsts.WorkloadNamespaceLabel: namespace,
			k8sconsts.WorkloadKindLabel:      string(obj.Kind),
		})
		sourceList, err := sourcesClient.Sources(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return nil, err
		}
		if len(sourceList.Items) > 1 {
			return nil, odigosv1.ErrorTooManySources
		}
		if len(sourceList.Items) == 1 {
			workloadSources.Workload = &sourceList.Items[0]
		}
	}

	namespaceSelector := labels.SelectorFromSet(labels.Set{
		k8sconsts.WorkloadNameLabel:      namespace,
		k8sconsts.WorkloadNamespaceLabel: namespace,
		k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
	})
	namespaceSourceList, err := sourcesClient.Sources(namespace).List(ctx, metav1.ListOptions{LabelSelector: namespaceSelector.String()})
	if err != nil {
		return nil, err
	}
	if len(namespaceSourceList.Items) > 1 {
		return nil, odigosv1.ErrorTooManySources
	}
	if len(namespaceSourceList.Items) == 1 {
		workloadSources.Namespace = &namespaceSourceList.Items[0]
	}

	return workloadSources, nil
}
