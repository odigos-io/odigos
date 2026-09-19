package diagnose

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

const (
	dgNamespace = "odigos-system"
	dgAppNs     = "shop"
	dgRootDir   = "odigos_debug_bundle"
)

var (
	deploymentConfigGVR = schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
	argoRolloutGVR      = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}
)

// dgFile is one entry the code under test asked a Builder to write.
type dgFile struct {
	dir      string
	filename string
	data     []byte
	gzipped  bool
}

// dgBuilder records what a diagnose stage wrote instead of touching the filesystem,
// so a test can assert the exact bundle layout. failOn injects per-file write errors.
type dgBuilder struct {
	mu     sync.Mutex
	files  []dgFile
	stats  BuilderStats
	failOn func(dir, filename string) error
}

func newDgBuilder() *dgBuilder {
	return &dgBuilder{}
}

func (b *dgBuilder) AddFile(dir, filename string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failOn != nil {
		if err := b.failOn(dir, filename); err != nil {
			return err
		}
	}
	b.files = append(b.files, dgFile{dir: dir, filename: filename, data: data})
	b.stats.TotalSize += int64(len(data))
	b.stats.FileCount++
	return nil
}

func (b *dgBuilder) AddFileGzipped(dir, filename string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failOn != nil {
		if failErr := b.failOn(dir, filename); failErr != nil {
			return failErr
		}
	}
	b.files = append(b.files, dgFile{dir: dir, filename: filename, data: data, gzipped: true})
	b.stats.TotalSize += int64(len(data))
	b.stats.FileCount++
	return nil
}

func (b *dgBuilder) GetStats() BuilderStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stats
}

// paths returns every written "<dir>/<filename>", sorted, with duplicates kept so a
// test can see that the same file was written twice.
func (b *dgBuilder) paths() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.files))
	for _, f := range b.files {
		out = append(out, path.Join(f.dir, f.filename))
	}
	sort.Strings(out)
	return out
}

func (b *dgBuilder) count(p string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, f := range b.files {
		if path.Join(f.dir, f.filename) == p {
			n++
		}
	}
	return n
}

func (b *dgBuilder) body(t *testing.T, p string) []byte {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, f := range b.files {
		if path.Join(f.dir, f.filename) == p {
			return f.data
		}
	}
	require.FailNowf(t, "file not written", "%q not in %v", p, b.files)
	return nil
}

func (b *dgBuilder) gzippedPaths() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []string
	for _, f := range b.files {
		if f.gzipped {
			out = append(out, path.Join(f.dir, f.filename))
		}
	}
	sort.Strings(out)
	return out
}

// dgRESTClientset returns a real clientset talking to handler. The typed fake clientset
// returns a nil RESTClient(), so the pod-proxy reads in metrics.go / profiles.go can only
// be driven through a real client. ContentType must be pinned to JSON because client-go
// negotiates protobuf for writes by default.
func dgRESTClientset(t *testing.T, handler http.HandlerFunc) *kubernetes.Clientset {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := kubernetes.NewForConfig(&rest.Config{
		Host:          srv.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json"},
		// A whole collection makes more requests than the default 5 QPS allows, and
		// waiting out the client-side limiter would only make the suite slow.
		QPS: -1,
	})
	require.NoError(t, err)
	return client
}

// dgCluster is a minimal apiserver for the reads diagnose performs. The typed fake
// clientset cannot serve the pod-proxy reads (its RESTClient() is nil), so a whole
// RunDiagnose can only be driven against a real client.
type dgCluster struct {
	pods        []corev1.Pod
	deployments []appsv1.Deployment
	daemonsets  []appsv1.DaemonSet
	configMaps  []corev1.ConfigMap

	mu       sync.Mutex
	requests []string
}

func (c *dgCluster) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		c.requests = append(c.requests, r.URL.RequestURI())
		c.mu.Unlock()

		p := r.URL.Path
		namespace := dgPathNamespace(p)
		switch {
		case strings.Contains(p, "/proxy/"):
			_, _ = io.WriteString(w, "proxied "+p)
		case strings.HasSuffix(p, "/log"):
			_, _ = io.WriteString(w, "log line for "+r.URL.Query().Get("container"))
		case strings.HasSuffix(p, "/pods"):
			dgWritePodList(t, w, dgSelectPods(t, dgPodsIn(c.pods, namespace), r.URL.Query().Get("labelSelector"))...)
		case strings.HasSuffix(p, "/deployments"):
			dgWriteJSON(t, w, &appsv1.DeploymentList{
				TypeMeta: metav1.TypeMeta{Kind: "DeploymentList", APIVersion: "apps/v1"},
				Items:    dgDeploymentsIn(c.deployments, namespace),
			})
		case strings.HasSuffix(p, "/daemonsets"):
			dgWriteJSON(t, w, &appsv1.DaemonSetList{
				TypeMeta: metav1.TypeMeta{Kind: "DaemonSetList", APIVersion: "apps/v1"},
				Items:    dgDaemonSetsIn(c.daemonsets, namespace),
			})
		case strings.HasSuffix(p, "/configmaps"):
			dgWriteJSON(t, w, &corev1.ConfigMapList{
				TypeMeta: metav1.TypeMeta{Kind: "ConfigMapList", APIVersion: "v1"},
				Items:    dgConfigMapsIn(c.configMaps, namespace),
			})
		case strings.HasSuffix(p, "/statefulsets"):
			dgWriteJSON(t, w, &appsv1.StatefulSetList{
				TypeMeta: metav1.TypeMeta{Kind: "StatefulSetList", APIVersion: "apps/v1"},
			})
		case strings.HasSuffix(p, "/cronjobs"):
			dgWriteJSON(t, w, &batchv1.CronJobList{
				TypeMeta: metav1.TypeMeta{Kind: "CronJobList", APIVersion: "batch/v1"},
			})
		case strings.Contains(p, "/deployments/"):
			if d := dgFindDeployment(c.deployments, path.Base(p)); d != nil {
				dgWriteJSON(t, w, d)
				return
			}
			dgWriteNotFound(w, "deployments", path.Base(p))
		case strings.Contains(p, "/daemonsets/"):
			if d := dgFindDaemonSet(c.daemonsets, path.Base(p)); d != nil {
				dgWriteJSON(t, w, d)
				return
			}
			dgWriteNotFound(w, "daemonsets", path.Base(p))
		default:
			dgWriteNotFound(w, "unknown", p)
		}
	}
}

func (c *dgCluster) recordedRequests() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.requests...)
}

func dgPathNamespace(urlPath string) string {
	segments := strings.Split(urlPath, "/")
	for i, segment := range segments {
		if segment == "namespaces" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}

// The dgXxxIn helpers keep only the objects the requested namespace holds, so a stage
// that reads the wrong namespace comes back empty instead of silently succeeding.
func dgPodsIn(items []corev1.Pod, namespace string) []corev1.Pod {
	var out []corev1.Pod
	for _, item := range items {
		if item.Namespace == namespace {
			out = append(out, item)
		}
	}
	return out
}

func dgDeploymentsIn(items []appsv1.Deployment, namespace string) []appsv1.Deployment {
	var out []appsv1.Deployment
	for _, item := range items {
		if item.Namespace == namespace {
			out = append(out, item)
		}
	}
	return out
}

func dgDaemonSetsIn(items []appsv1.DaemonSet, namespace string) []appsv1.DaemonSet {
	var out []appsv1.DaemonSet
	for _, item := range items {
		if item.Namespace == namespace {
			out = append(out, item)
		}
	}
	return out
}

func dgConfigMapsIn(items []corev1.ConfigMap, namespace string) []corev1.ConfigMap {
	var out []corev1.ConfigMap
	for _, item := range items {
		if item.Namespace == namespace {
			out = append(out, item)
		}
	}
	return out
}

func dgFindDeployment(items []appsv1.Deployment, name string) *appsv1.Deployment {
	for i := range items {
		if items[i].Name == name {
			out := items[i]
			out.TypeMeta = metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}
			return &out
		}
	}
	return nil
}

func dgFindDaemonSet(items []appsv1.DaemonSet, name string) *appsv1.DaemonSet {
	for i := range items {
		if items[i].Name == name {
			out := items[i]
			out.TypeMeta = metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"}
			return &out
		}
	}
	return nil
}

func dgSelectPods(t *testing.T, pods []corev1.Pod, selector string) []corev1.Pod {
	t.Helper()
	parsed, err := labels.Parse(selector)
	require.NoError(t, err)
	var out []corev1.Pod
	for i := range pods {
		if parsed.Matches(labels.Set(pods[i].Labels)) {
			out = append(out, pods[i])
		}
	}
	return out
}

func dgWriteJSON(t *testing.T, w http.ResponseWriter, obj interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(obj))
}

func dgWriteNotFound(w http.ResponseWriter, resource, name string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(&metav1.Status{
		TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
		Status:   metav1.StatusFailure,
		Code:     http.StatusNotFound,
		Reason:   metav1.StatusReasonNotFound,
		Message:  resource + " " + name + " not found",
		Details:  &metav1.StatusDetails{Name: name, Kind: resource},
	})
}

func dgWritePodList(t *testing.T, w http.ResponseWriter, pods ...corev1.Pod) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(&corev1.PodList{
		TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"},
		Items:    pods,
	}))
}

func dgDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		deploymentConfigGVR:                 "DeploymentConfigList",
		argoRolloutGVR:                      "RolloutList",
		odigosGVR("destinations"):           "DestinationList",
		odigosGVR("sources"):                "SourceList",
		odigosGVR("instrumentationconfigs"): "InstrumentationConfigList",
		actionsGVR("piimaskings"):           "PiiMaskingList",
	}, objects...)
}

func odigosGVR(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: odigosGroupName, Version: "v1alpha1", Resource: resource}
}

func actionsGVR(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: actionsGroupName, Version: "v1alpha1", Resource: resource}
}

func dgOdigosClient(sources ...odigosv1.Source) odigosv1alpha1.OdigosV1alpha1Interface {
	objects := make([]runtime.Object, 0, len(sources))
	for i := range sources {
		objects = append(objects, &sources[i])
	}
	return odigosfake.NewSimpleClientset(objects...).OdigosV1alpha1()
}

// dgFailingOdigosClient cannot list Sources, as happens when the caller's role is not
// allowed to read them cluster-wide.
func dgFailingOdigosClient() odigosv1alpha1.OdigosV1alpha1Interface {
	client := odigosfake.NewSimpleClientset()
	client.PrependReactor("list", "sources", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: odigosGroupName, Resource: "sources"}, "", fmt.Errorf("no access"))
	})
	return client.OdigosV1alpha1()
}

// dgSource builds a Source CRD for a workload, the way the instrumentor writes it.
func dgSource(namespace string, kind k8sconsts.WorkloadKind, workloadNamespace, name string) odigosv1.Source {
	return odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(kind) + "-" + name,
			Namespace: namespace,
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{Kind: kind, Name: name, Namespace: workloadNamespace},
		},
	}
}

func dgDeployment(namespace, name string, selector map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, ManagedFields: dgManagedFields()},
		Spec:       appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: selector}},
	}
}

func dgDaemonSet(namespace, name string, selector map[string]string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, ManagedFields: dgManagedFields()},
		Spec:       appsv1.DaemonSetSpec{Selector: &metav1.LabelSelector{MatchLabels: selector}},
	}
}

func dgStatefulSet(namespace, name string, selector map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, ManagedFields: dgManagedFields()},
		Spec:       appsv1.StatefulSetSpec{Selector: &metav1.LabelSelector{MatchLabels: selector}},
	}
}

func dgCronJob(namespace, name string, selector map[string]string) *batchv1.CronJob {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, ManagedFields: dgManagedFields()},
	}
	if selector != nil {
		cj.Spec.JobTemplate.Spec.Selector = &metav1.LabelSelector{MatchLabels: selector}
	}
	return cj
}

func dgJob(namespace, name string, selector map[string]string) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, ManagedFields: dgManagedFields()},
	}
	if selector != nil {
		job.Spec.Selector = &metav1.LabelSelector{MatchLabels: selector}
	}
	return job
}

// dgDeploymentConfigObject is an OpenShift DeploymentConfig, whose selector is a flat
// map under spec.selector rather than spec.selector.matchLabels.
func dgDeploymentConfigObject(namespace, name string, selector map[string]string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.openshift.io/v1",
		"kind":       "DeploymentConfig",
		"metadata": map[string]interface{}{
			"name":          name,
			"namespace":     namespace,
			"managedFields": []interface{}{map[string]interface{}{"manager": "openshift-controller-manager"}},
		},
		"spec": map[string]interface{}{"selector": dgStringMap(selector)},
	}}
}

func dgArgoRolloutObject(namespace, name string, selector map[string]string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Rollout",
		"metadata": map[string]interface{}{
			"name":          name,
			"namespace":     namespace,
			"managedFields": []interface{}{map[string]interface{}{"manager": "argo-rollouts"}},
		},
		"spec": map[string]interface{}{"selector": map[string]interface{}{"matchLabels": dgStringMap(selector)}},
	}}
}

func dgStringMap(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func dgPod(namespace, name string, labels map[string]string, containers ...string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:     namespace,
			Name:          name,
			Labels:        labels,
			ManagedFields: dgManagedFields(),
		},
		Spec: corev1.PodSpec{NodeName: "node-a"},
	}
	for _, c := range containers {
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: c})
	}
	return pod
}

// dgManagedFields is the noise every collected object must be stripped of before it
// reaches a support bundle.
func dgManagedFields() []metav1.ManagedFieldsEntry {
	return []metav1.ManagedFieldsEntry{{Manager: "kube-controller-manager", Operation: metav1.ManagedFieldsOperationUpdate}}
}

func dgClientset(objects ...runtime.Object) *fake.Clientset {
	return fake.NewSimpleClientset(objects...)
}

// dgClients is the set of clients RunDiagnose takes.
type dgClients struct {
	kube    *fake.Clientset
	dynamic *dynamicfake.FakeDynamicClient
	odigos  odigosv1alpha1.OdigosV1alpha1Interface
}

func dgEmptyCluster() *dgClients {
	return &dgClients{kube: dgClientset(), dynamic: dgDynamicClient(), odigos: dgOdigosClient()}
}
