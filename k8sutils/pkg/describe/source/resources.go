package source

import (
	"context"

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
	InstrumentedApplication  *odigosv1.InstrumentedApplication
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

	ia, err := odigosClient.InstrumentedApplications(workloadNs).Get(ctx, runtimeObjectName, metav1.GetOptions{})
	if err == nil {
		sourceResources.InstrumentedApplication = ia
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

	podLabelSelector := metav1.FormatLabelSelector(workloadObj.LabelSelector)
	if err != nil {
		// if pod info cannot be extracted, it is an unrecoverable error
		return nil, err
	}
	pods, err := kubeClient.CoreV1().Pods(workloadNs).List(ctx, metav1.ListOptions{LabelSelector: podLabelSelector})
	if err == nil {
		sourceResources.Pods = pods
	} else {
		return nil, err
	}

	return &sourceResources, nil
}
