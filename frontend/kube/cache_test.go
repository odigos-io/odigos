package kube

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func jsonSize(t *testing.T, obj interface{}) int {
	t.Helper()
	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("failed to marshal object: %v", err)
	}
	return len(data)
}

// --- Bloated object generators ---

func newBloatedDeployment(ns string, i int) *appsv1.Deployment {
	labels := make(map[string]string, 15)
	for j := range 15 {
		labels[fmt.Sprintf("label-%d", j)] = fmt.Sprintf("value-%d-with-some-extra-padding-to-simulate-real-labels", j)
	}
	annotations := make(map[string]string, 10)
	for j := range 10 {
		annotations[fmt.Sprintf("annotation-%d", j)] = fmt.Sprintf("annotation-value-%d-with-padding", j)
	}
	annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"` + fmt.Sprintf("deploy-%d", i) + `","namespace":"` + ns + `","labels":{"app":"myapp","version":"v1","team":"backend","env":"production","tier":"api","component":"server","release":"stable","managed-by":"helm","chart":"myapp-1.2.3","heritage":"Helm"}},"spec":{"replicas":3,"selector":{"matchLabels":{"app":"myapp"}},"template":{"metadata":{"labels":{"app":"myapp","version":"v1"}},"spec":{"containers":[{"name":"main","image":"myapp:latest","ports":[{"containerPort":8080}],"env":[{"name":"DB_HOST","value":"postgres"},{"name":"DB_PORT","value":"5432"},{"name":"DB_NAME","value":"mydb"},{"name":"CACHE_HOST","value":"redis"},{"name":"LOG_LEVEL","value":"info"}],"resources":{"requests":{"cpu":"100m","memory":"256Mi"},"limits":{"cpu":"500m","memory":"512Mi"}},"volumeMounts":[{"name":"config","mountPath":"/etc/config"},{"name":"secrets","mountPath":"/etc/secrets"}]},{"name":"sidecar","image":"envoy:latest"}],"volumes":[{"name":"config","configMap":{"name":"myapp-config"}},{"name":"secrets","secret":{"secretName":"myapp-secrets"}}]}}}}`

	envVars := make([]corev1.EnvVar, 10)
	for j := range 10 {
		envVars[j] = corev1.EnvVar{Name: fmt.Sprintf("ENV_%d", j), Value: fmt.Sprintf("value-%d", j)}
	}

	containers := make([]corev1.Container, 3)
	for j := range 3 {
		containers[j] = corev1.Container{
			Name:  fmt.Sprintf("container-%d", j),
			Image: fmt.Sprintf("myimage:%d.%d", i, j),
			Env:   envVars,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data", MountPath: "/data"},
				{Name: "config", MountPath: "/etc/config"},
			},
		}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("deploy-%d", i),
			Namespace:   ns,
			UID:         types.UID(fmt.Sprintf("uid-%d", i)),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: containers,
					Volumes: []corev1.Volume{
						{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "my-config"}}}},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     3,
			AvailableReplicas: 3,
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue, Reason: "MinimumReplicasAvailable"},
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
			},
		},
	}
}

func newBloatedStatefulSet(ns string, i int) *appsv1.StatefulSet {
	labels := make(map[string]string, 15)
	for j := range 15 {
		labels[fmt.Sprintf("label-%d", j)] = fmt.Sprintf("value-%d-with-extra-padding", j)
	}
	annotations := make(map[string]string, 10)
	for j := range 10 {
		annotations[fmt.Sprintf("annotation-%d", j)] = fmt.Sprintf("annotation-value-%d-with-padding", j)
	}
	annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"kind":"StatefulSet","spec":{"replicas":3,"volumeClaimTemplates":[{"metadata":{"name":"data"},"spec":{"accessModes":["ReadWriteOnce"],"resources":{"requests":{"storage":"10Gi"}}}}]}}`

	envVars := make([]corev1.EnvVar, 10)
	for j := range 10 {
		envVars[j] = corev1.EnvVar{Name: fmt.Sprintf("ENV_%d", j), Value: fmt.Sprintf("value-%d", j)}
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("sts-%d", i),
			Namespace:   ns,
			UID:         types.UID(fmt.Sprintf("uid-sts-%d", i)),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main", Image: "myapp:latest", Env: envVars},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{ObjectMeta: metav1.ObjectMeta{Name: "data"}, Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				}},
			},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 3,
		},
	}
}

func newBloatedDaemonSet(ns string, i int) *appsv1.DaemonSet {
	labels := make(map[string]string, 15)
	for j := range 15 {
		labels[fmt.Sprintf("label-%d", j)] = fmt.Sprintf("value-%d-with-extra-padding", j)
	}
	annotations := make(map[string]string, 10)
	for j := range 10 {
		annotations[fmt.Sprintf("annotation-%d", j)] = fmt.Sprintf("annotation-value-%d-with-padding", j)
	}

	envVars := make([]corev1.EnvVar, 10)
	for j := range 10 {
		envVars[j] = corev1.EnvVar{Name: fmt.Sprintf("ENV_%d", j), Value: fmt.Sprintf("value-%d", j)}
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("ds-%d", i),
			Namespace:   ns,
			UID:         types.UID(fmt.Sprintf("uid-ds-%d", i)),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "agent", Image: "agent:latest", Env: envVars},
					},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			NumberReady:            5,
			DesiredNumberScheduled: 5,
			CurrentNumberScheduled: 5,
			NumberAvailable:        5,
		},
	}
}

func newBloatedCronJob(ns string, i int) *batchv1.CronJob {
	labels := make(map[string]string, 15)
	for j := range 15 {
		labels[fmt.Sprintf("label-%d", j)] = fmt.Sprintf("value-%d-with-extra-padding", j)
	}
	annotations := make(map[string]string, 10)
	for j := range 10 {
		annotations[fmt.Sprintf("annotation-%d", j)] = fmt.Sprintf("annotation-value-%d-with-padding", j)
	}

	envVars := make([]corev1.EnvVar, 10)
	for j := range 10 {
		envVars[j] = corev1.EnvVar{Name: fmt.Sprintf("ENV_%d", j), Value: fmt.Sprintf("value-%d", j)}
	}

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("cj-%d", i),
			Namespace:   ns,
			UID:         types.UID(fmt.Sprintf("uid-cj-%d", i)),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "job", Image: "batch:latest", Env: envVars},
							},
						},
					},
				},
			},
		},
		Status: batchv1.CronJobStatus{
			Active: []corev1.ObjectReference{
				{Name: "cj-0-12345", Namespace: ns},
				{Name: "cj-0-12346", Namespace: ns},
				{Name: "cj-0-12347", Namespace: ns},
			},
		},
	}
}

func newBloatedPod(ns string, i int) *corev1.Pod {
	labels := make(map[string]string, 15)
	for j := range 14 {
		labels[fmt.Sprintf("label-%d", j)] = fmt.Sprintf("value-%d-with-extra-padding", j)
	}
	labels[k8sconsts.OdigosAgentsMetaHashLabel] = "abc123def456"

	envVars := make([]corev1.EnvVar, 10)
	for j := range 9 {
		envVars[j] = corev1.EnvVar{Name: fmt.Sprintf("ENV_%d", j), Value: fmt.Sprintf("value-%d", j)}
	}
	envVars[9] = corev1.EnvVar{Name: k8sconsts.OdigosEnvVarDistroName, Value: "odigos-opentelemetry-go"}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pod-%d", i),
			Namespace: ns,
			UID:       types.UID(fmt.Sprintf("uid-pod-%d", i)),
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: fmt.Sprintf("deploy-%d-abc", i), UID: types.UID(fmt.Sprintf("uid-rs-%d", i))},
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			NodeName: "worker-1",
			Containers: []corev1.Container{
				{Name: "main", Image: "myapp:latest", Env: envVars, Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
					Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("512Mi")},
				}},
				{Name: "sidecar", Image: "envoy:latest", Env: envVars},
			},
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "main", Ready: true, RestartCount: 0, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{Name: "sidecar", Ready: true, RestartCount: 0, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{Name: "init", Ready: true, RestartCount: 0, State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
			},
		},
	}
}

// --- Per-transform size tests ---

func TestDeploymentsTransformSize(t *testing.T) {
	dep := newBloatedDeployment("test-ns", 0)
	bloatedSize := jsonSize(t, dep)
	if bloatedSize < 5000 {
		t.Fatalf("bloated deployment is only %d bytes (generator not bloated enough)", bloatedSize)
	}

	result, err := deploymentsTransformFunc(dep)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	transformedSize := jsonSize(t, result)
	t.Logf("Deployment: bloated=%d bytes, transformed=%d bytes (%.0f%% reduction)", bloatedSize, transformedSize, 100*(1-float64(transformedSize)/float64(bloatedSize)))

	if transformedSize > 500 {
		t.Fatalf("transformed deployment is %d bytes, exceeds 500 byte budget", transformedSize)
	}

	// Correctness spot-checks
	transformed := result.(*appsv1.Deployment)
	if transformed.Name != dep.Name || transformed.Namespace != dep.Namespace {
		t.Fatal("Name/Namespace not preserved")
	}
	if transformed.UID != dep.UID {
		t.Fatal("UID not preserved")
	}
	if transformed.Status.ReadyReplicas != dep.Status.ReadyReplicas {
		t.Fatal("ReadyReplicas not preserved")
	}
	if transformed.Status.AvailableReplicas != dep.Status.AvailableReplicas {
		t.Fatal("AvailableReplicas not preserved")
	}
	if len(transformed.Labels) != 0 {
		t.Fatalf("expected labels stripped, got %d", len(transformed.Labels))
	}
	if len(transformed.Annotations) != 0 {
		t.Fatalf("expected annotations stripped, got %d", len(transformed.Annotations))
	}
}

func TestStatefulSetsTransformSize(t *testing.T) {
	ss := newBloatedStatefulSet("test-ns", 0)
	bloatedSize := jsonSize(t, ss)
	if bloatedSize < 1500 {
		t.Fatalf("bloated statefulset is only %d bytes", bloatedSize)
	}

	result, err := statefulSetsTransformFunc(ss)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	transformedSize := jsonSize(t, result)
	t.Logf("StatefulSet: bloated=%d bytes, transformed=%d bytes (%.0f%% reduction)", bloatedSize, transformedSize, 100*(1-float64(transformedSize)/float64(bloatedSize)))

	if transformedSize > 500 {
		t.Fatalf("transformed statefulset is %d bytes, exceeds 500 byte budget", transformedSize)
	}

	transformed := result.(*appsv1.StatefulSet)
	if transformed.Name != ss.Name || transformed.Namespace != ss.Namespace {
		t.Fatal("Name/Namespace not preserved")
	}
	if transformed.Status.ReadyReplicas != ss.Status.ReadyReplicas {
		t.Fatal("ReadyReplicas not preserved")
	}
}

func TestDaemonSetsTransformSize(t *testing.T) {
	ds := newBloatedDaemonSet("test-ns", 0)
	bloatedSize := jsonSize(t, ds)
	if bloatedSize < 1500 {
		t.Fatalf("bloated daemonset is only %d bytes", bloatedSize)
	}

	result, err := daemonSetsTransformFunc(ds)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	transformedSize := jsonSize(t, result)
	t.Logf("DaemonSet: bloated=%d bytes, transformed=%d bytes (%.0f%% reduction)", bloatedSize, transformedSize, 100*(1-float64(transformedSize)/float64(bloatedSize)))

	if transformedSize > 500 {
		t.Fatalf("transformed daemonset is %d bytes, exceeds 500 byte budget", transformedSize)
	}

	transformed := result.(*appsv1.DaemonSet)
	if transformed.Name != ds.Name || transformed.Namespace != ds.Namespace {
		t.Fatal("Name/Namespace not preserved")
	}
	if transformed.Status.NumberReady != ds.Status.NumberReady {
		t.Fatal("NumberReady not preserved")
	}
}

func TestCronJobsTransformSize(t *testing.T) {
	cj := newBloatedCronJob("test-ns", 0)
	bloatedSize := jsonSize(t, cj)
	if bloatedSize < 1500 {
		t.Fatalf("bloated cronjob is only %d bytes", bloatedSize)
	}

	result, err := cronJobsTransformFunc(cj)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	transformedSize := jsonSize(t, result)
	t.Logf("CronJob: bloated=%d bytes, transformed=%d bytes (%.0f%% reduction)", bloatedSize, transformedSize, 100*(1-float64(transformedSize)/float64(bloatedSize)))

	if transformedSize > 1000 {
		t.Fatalf("transformed cronjob is %d bytes, exceeds 1000 byte budget", transformedSize)
	}

	transformed := result.(*batchv1.CronJob)
	if transformed.Name != cj.Name || transformed.Namespace != cj.Namespace {
		t.Fatal("Name/Namespace not preserved")
	}
	if len(transformed.Status.Active) != len(cj.Status.Active) {
		t.Fatal("Active refs not preserved")
	}
}

func TestPodsTransformSize(t *testing.T) {
	pod := newBloatedPod("test-ns", 0)
	bloatedSize := jsonSize(t, pod)
	if bloatedSize < 1500 {
		t.Fatalf("bloated pod is only %d bytes", bloatedSize)
	}

	result, err := podsTransformFunc(pod)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	transformedSize := jsonSize(t, result)
	t.Logf("Pod: bloated=%d bytes, transformed=%d bytes (%.0f%% reduction)", bloatedSize, transformedSize, 100*(1-float64(transformedSize)/float64(bloatedSize)))

	if transformedSize > 3000 {
		t.Fatalf("transformed pod is %d bytes, exceeds 3000 byte budget", transformedSize)
	}

	// Correctness spot-checks
	transformed := result.(*corev1.Pod)
	if transformed.Name != pod.Name || transformed.Namespace != pod.Namespace {
		t.Fatal("Name/Namespace not preserved")
	}
	if transformed.CreationTimestamp.IsZero() {
		t.Fatal("CreationTimestamp not preserved")
	}
	if len(transformed.OwnerReferences) != 1 {
		t.Fatalf("OwnerReferences not preserved, got %d", len(transformed.OwnerReferences))
	}
	if transformed.Spec.NodeName != "worker-1" {
		t.Fatalf("NodeName not preserved, got %q", transformed.Spec.NodeName)
	}

	// Only the OdigosAgentsMetaHashLabel should be kept
	if len(transformed.Labels) != 1 {
		t.Fatalf("expected 1 label (OdigosAgentsMetaHashLabel), got %d", len(transformed.Labels))
	}
	if _, ok := transformed.Labels[k8sconsts.OdigosAgentsMetaHashLabel]; !ok {
		t.Fatal("OdigosAgentsMetaHashLabel not preserved")
	}

	// Each container should only have ODIGOS_DISTRO_NAME env var
	for _, c := range transformed.Spec.Containers {
		for _, env := range c.Env {
			if env.Name != k8sconsts.OdigosEnvVarDistroName {
				t.Fatalf("unexpected env var %q in container %q", env.Name, c.Name)
			}
		}
	}

	// ContainerStatuses preserved
	if len(transformed.Status.ContainerStatuses) != 2 {
		t.Fatalf("expected 2 ContainerStatuses, got %d", len(transformed.Status.ContainerStatuses))
	}
	if len(transformed.Status.InitContainerStatuses) != 1 {
		t.Fatalf("expected 1 InitContainerStatus, got %d", len(transformed.Status.InitContainerStatuses))
	}
}

func TestTransformWrongType(t *testing.T) {
	wrongObj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm"}}

	transforms := map[string]func(interface{}) (interface{}, error){
		"deployments":  deploymentsTransformFunc,
		"statefulsets":  statefulSetsTransformFunc,
		"daemonsets":    daemonSetsTransformFunc,
		"cronjobs":      cronJobsTransformFunc,
		"pods":          podsTransformFunc,
	}

	for name, fn := range transforms {
		_, err := fn(wrongObj)
		if err == nil {
			t.Errorf("%s: expected error for wrong type, got nil", name)
		}
	}
}

func TestTransformScaleMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	const count = 10000

	// Force GC and measure baseline
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	transformed := make([]*appsv1.Deployment, count)
	for i := range count {
		dep := newBloatedDeployment("test-ns", i)
		result, err := deploymentsTransformFunc(dep)
		if err != nil {
			t.Fatalf("transform failed at %d: %v", i, err)
		}
		transformed[i] = result.(*appsv1.Deployment)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	heapDelta := after.HeapAlloc - before.HeapAlloc
	heapDeltaMB := float64(heapDelta) / 1024 / 1024

	t.Logf("Scale test: %d transformed deployments, heap delta: %.1f MB", count, heapDeltaMB)

	// Transformed objects should use ~2-5 MB for 10K items.
	// Without transforms, full objects would use ~200+ MB.
	const maxMB = 50
	if heapDeltaMB > maxMB {
		t.Fatalf("heap delta %.1f MB exceeds %d MB budget (transforms may be broken)", heapDeltaMB, maxMB)
	}

	// Keep reference alive so GC doesn't collect before measurement
	_ = transformed[count-1].Name
}
