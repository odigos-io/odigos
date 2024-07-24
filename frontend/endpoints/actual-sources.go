package endpoints

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetActualSources(ctx context.Context, odigosns string) []ThinSource {

	effectiveInstrumentedSources := map[SourceID]ThinSource{}

	var (
		items                    []GetApplicationItem
		instrumentedApplications *v1alpha1.InstrumentedApplicationList
	)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		relevantNamespaces, err := getRelevantNameSpaces(ctx, odigosns)
		if err != nil {
			return err
		}
		nsInstrumentedMap := map[string]*bool{}
		for _, ns := range relevantNamespaces {
			nsInstrumentedMap[ns.Name] = isObjectLabeledForInstrumentation(ns.ObjectMeta)
		}
		// get all the applications in all the namespaces,
		// passing an empty string here is more efficient compared to iterating over the namespaces
		// since it will make a single request per workload type to the k8s api server
		items, err = getApplicationsInNamespace(ctx, "", nsInstrumentedMap)
		return err
	})

	g.Go(func() error {
		var err error
		instrumentedApplications, err = kube.DefaultClient.OdigosClient.InstrumentedApplications("").List(ctx, metav1.ListOptions{})
		return err
	})

	if err := g.Wait(); err != nil {
		return nil
	}

	for _, item := range items {
		if item.nsItem.InstrumentationEffective {
			id := SourceID{Namespace: item.namespace, Kind: string(item.nsItem.Kind), Name: item.nsItem.Name}
			effectiveInstrumentedSources[id] = ThinSource{
				NumberOfRunningInstances: item.nsItem.Instances,
				SourceID:                 id,
			}
		}
	}

	sourcesResult := []ThinSource{}
	// go over the instrumented applications and update the languages of the effective sources.
	// Not all effective sources necessarily have a corresponding instrumented application,
	// it may take some time for the instrumented application to be created. In that case the languages
	// slice will be empty.
	for _, app := range instrumentedApplications.Items {
		thinSource := k8sInstrumentedAppToThinSource(&app)
		if source, ok := effectiveInstrumentedSources[thinSource.SourceID]; ok {
			source.IaDetails = thinSource.IaDetails
			effectiveInstrumentedSources[thinSource.SourceID] = source
		}
	}

	for _, source := range effectiveInstrumentedSources {
		sourcesResult = append(sourcesResult, source)
	}

	return sourcesResult

}

func GetActualSource(ctx context.Context, ns string, kind string, name string) (*Source, error) {

	k8sObjectName := workload.GetRuntimeObjectName(name, kind)

	owner, numberOfRunningInstances := getWorkload(ctx, ns, kind, name)
	if owner == nil {
		return nil, fmt.Errorf("owner not found")
	}
	ownerAnnotations := owner.GetAnnotations()
	var reportedName string
	if ownerAnnotations != nil {
		reportedName = ownerAnnotations[consts.OdigosReportedNameAnnotation]
	}

	ts := ThinSource{
		SourceID: SourceID{
			Namespace: ns,
			Kind:      kind,
			Name:      name,
		},
		NumberOfRunningInstances: numberOfRunningInstances,
	}

	instrumentedApplication, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(ns).Get(ctx, k8sObjectName, metav1.GetOptions{})
	if err == nil {
		// valid instrumented application, grab the runtime details
		ts.IaDetails = k8sInstrumentedAppToThinSource(instrumentedApplication).IaDetails
		// potentially add a condition for healthy instrumentation instances
		err = addHealthyInstrumentationInstancesCondition(ctx, instrumentedApplication, &ts)
		if err != nil {
			return nil, err
		}
	}

	return &Source{
		ThinSource:   ts,
		ReportedName: reportedName,
	}, nil
}

func getWorkload(c context.Context, ns string, kind string, name string) (metav1.Object, int) {
	switch kind {
	case "Deployment":
		deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, 0
		}
		return deployment, int(deployment.Status.AvailableReplicas)
	case "StatefulSet":
		statefulSet, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, 0
		}
		return statefulSet, int(statefulSet.Status.ReadyReplicas)
	case "DaemonSet":
		daemonSet, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, 0
		}
		return daemonSet, int(daemonSet.Status.NumberReady)
	default:
		return nil, 0
	}
}
