package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func httpGetProbe(path string) *v1.Probe {
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(8080),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
	}
}

func tcpProbe(port int32) *v1.Probe {
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			TCPSocket: &v1.TCPSocketAction{
				Port: intstr.FromInt32(port),
			},
		},
	}
}

func fullContainer(name string) v1.Container {
	return v1.Container{
		Name:    name,
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Args:    []string{"--config", "/etc/nginx.conf"},
		Env: []v1.EnvVar{
			{Name: "FOO", Value: "bar"},
		},
		Ports: []v1.ContainerPort{
			{ContainerPort: 80},
		},
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		VolumeMounts: []v1.VolumeMount{
			{Name: "data", MountPath: "/data"},
		},
		LivenessProbe:  httpGetProbe("/healthz"),
		ReadinessProbe: httpGetProbe("/ready"),
		StartupProbe:   tcpProbe(8080),
		SecurityContext: &v1.SecurityContext{
			RunAsNonRoot: boolPtr(true),
		},
	}
}

func fullPodSpec() v1.PodSpec {
	return v1.PodSpec{
		Containers: []v1.Container{
			fullContainer("app"),
			fullContainer("sidecar"),
		},
		InitContainers: []v1.Container{
			{Name: "init", Image: "busybox"},
		},
		Volumes: []v1.Volume{
			{Name: "data", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
		},
		ServiceAccountName: "my-sa",
		NodeSelector:       map[string]string{"zone": "us-east"},
		Tolerations: []v1.Toleration{
			{Key: "key1", Operator: v1.TolerationOpEqual, Value: "val1"},
		},
	}
}

func workloadAnnotations() map[string]string {
	return map[string]string{
		"kubernetes.io/name":      "keep-this",
		"odigos.io/enabled":       "true",
		"app.example.com/version": "should-be-stripped",
		"helm.sh/release":         "should-be-stripped",
	}
}

func templateAnnotations() map[string]string {
	return map[string]string{
		"kubernetes.io/restartedAt": "2025-01-01T00:00:00Z",
		"prometheus.io/scrape":      "true",
	}
}

func boolPtr(b bool) *bool { return &b }

func assertStrippedContainers(t *testing.T, containers []v1.Container) {
	t.Helper()
	assert.Len(t, containers, 2)

	for _, c := range containers {
		assert.NotEmpty(t, c.Name)
		assert.Empty(t, c.Image)
		assert.Nil(t, c.Command)
		assert.Nil(t, c.Args)
		assert.Nil(t, c.Env)
		assert.Nil(t, c.Ports)
		assert.Empty(t, c.Resources.Limits)
		assert.Empty(t, c.Resources.Requests)
		assert.Nil(t, c.VolumeMounts)
		assert.Nil(t, c.SecurityContext)
	}

	app := containers[0]
	assert.Equal(t, "app", app.Name)
	assert.NotNil(t, app.LivenessProbe)
	assert.Equal(t, "/healthz", app.LivenessProbe.HTTPGet.Path)
	assert.NotNil(t, app.ReadinessProbe)
	assert.Equal(t, "/ready", app.ReadinessProbe.HTTPGet.Path)
	assert.NotNil(t, app.StartupProbe)
	assert.NotNil(t, app.StartupProbe.TCPSocket)

	sidecar := containers[1]
	assert.Equal(t, "sidecar", sidecar.Name)
	assert.NotNil(t, sidecar.LivenessProbe)
	assert.NotNil(t, sidecar.ReadinessProbe)
	assert.NotNil(t, sidecar.StartupProbe)
}

func assertStrippedPodSpec(t *testing.T, spec v1.PodSpec) {
	t.Helper()
	assert.Nil(t, spec.InitContainers)
	assert.Nil(t, spec.Volumes)
	assert.Empty(t, spec.ServiceAccountName)
	assert.Nil(t, spec.NodeSelector)
	assert.Nil(t, spec.Tolerations)
}

func assertAnnotationsStripped(t *testing.T, annotations map[string]string) {
	t.Helper()
	assert.Contains(t, annotations, "kubernetes.io/name")
	assert.Contains(t, annotations, "odigos.io/enabled")
	assert.NotContains(t, annotations, "app.example.com/version")
	assert.NotContains(t, annotations, "helm.sh/release")
}

func assertTemplateAnnotationsStripped(t *testing.T, annotations map[string]string) {
	t.Helper()
	assert.Contains(t, annotations, "kubernetes.io/restartedAt")
	assert.NotContains(t, annotations, "prometheus.io/scrape")
}

func TestStripWorkloadSpecTemplate_Deployment(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dep", Namespace: "default",
			Annotations: workloadAnnotations(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: templateAnnotations()},
				Spec:       fullPodSpec(),
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 2,
			Replicas:          3,
		},
	}

	StripWorkloadSpecTemplate(dep)

	assertAnnotationsStripped(t, dep.Annotations)
	assertTemplateAnnotationsStripped(t, dep.Spec.Template.Annotations)
	assertStrippedContainers(t, dep.Spec.Template.Spec.Containers)
	assertStrippedPodSpec(t, dep.Spec.Template.Spec)

	assert.NotNil(t, dep.Spec.Selector, "Spec.Selector must be preserved")
	assert.Equal(t, int32(2), dep.Status.AvailableReplicas, "Status must be preserved")
	assert.Equal(t, int32(3), dep.Status.Replicas, "Status must be preserved")
}

func TestStripWorkloadSpecTemplate_StatefulSet(t *testing.T) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sts", Namespace: "default",
			Annotations: workloadAnnotations(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			ServiceName: "my-svc",
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: templateAnnotations()},
				Spec:       fullPodSpec(),
			},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 1,
			Replicas:      2,
		},
	}

	StripWorkloadSpecTemplate(sts)

	assertAnnotationsStripped(t, sts.Annotations)
	assertTemplateAnnotationsStripped(t, sts.Spec.Template.Annotations)
	assertStrippedContainers(t, sts.Spec.Template.Spec.Containers)
	assertStrippedPodSpec(t, sts.Spec.Template.Spec)

	assert.NotNil(t, sts.Spec.Selector, "Spec.Selector must be preserved")
	assert.Equal(t, int32(1), sts.Status.ReadyReplicas, "Status must be preserved")
	assert.Equal(t, int32(2), sts.Status.Replicas, "Status must be preserved")
}

func TestStripWorkloadSpecTemplate_DaemonSet(t *testing.T) {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ds", Namespace: "default",
			Annotations: workloadAnnotations(),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: templateAnnotations()},
				Spec:       fullPodSpec(),
			},
		},
		Status: appsv1.DaemonSetStatus{
			NumberReady:            3,
			DesiredNumberScheduled: 5,
		},
	}

	StripWorkloadSpecTemplate(ds)

	assertAnnotationsStripped(t, ds.Annotations)
	assertTemplateAnnotationsStripped(t, ds.Spec.Template.Annotations)
	assertStrippedContainers(t, ds.Spec.Template.Spec.Containers)
	assertStrippedPodSpec(t, ds.Spec.Template.Spec)

	assert.NotNil(t, ds.Spec.Selector, "Spec.Selector must be preserved")
	assert.Equal(t, int32(3), ds.Status.NumberReady, "Status must be preserved")
	assert.Equal(t, int32(5), ds.Status.DesiredNumberScheduled, "Status must be preserved")
}

func TestStripWorkloadSpecTemplate_EmptyContainers(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-containers"},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{{Name: "vol"}},
				},
			},
		},
	}

	StripWorkloadSpecTemplate(dep)

	assert.Empty(t, dep.Spec.Template.Spec.Containers)
	assert.Nil(t, dep.Spec.Template.Spec.Volumes, "PodSpec fields must be zeroed even when no containers")
}

func TestStripWorkloadSpecTemplate_SingleContainer_NoProbes(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "no-probes"},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "worker",
							Image: "python:3.11",
							Env:   []v1.EnvVar{{Name: "DEBUG", Value: "1"}},
						},
					},
				},
			},
		},
	}

	StripWorkloadSpecTemplate(dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	c := dep.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "worker", c.Name)
	assert.Empty(t, c.Image)
	assert.Nil(t, c.Env)
	assert.Nil(t, c.LivenessProbe)
	assert.Nil(t, c.ReadinessProbe)
	assert.Nil(t, c.StartupProbe)
}

func TestStripWorkloadSpecTemplate_UnsupportedType_Noop(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "standalone-pod",
			Annotations: map[string]string{"custom.io/test": "value"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "app", Image: "nginx"},
			},
		},
	}

	StripWorkloadSpecTemplate(pod)

	// Annotations on the object itself are still stripped (StripAnnotations runs for all types)
	assert.NotContains(t, pod.Annotations, "custom.io/test")
	// But the pod spec containers are untouched since Pod is not a recognized workload type
	assert.Equal(t, "nginx", pod.Spec.Containers[0].Image)
}

func TestStripWorkloadSpecTemplate_PreservesProbeDetails(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "probe-details"},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:           "app",
							LivenessProbe:  httpGetProbe("/live"),
							ReadinessProbe: httpGetProbe("/ready"),
							StartupProbe:   httpGetProbe("/start"),
						},
					},
				},
			},
		},
	}

	StripWorkloadSpecTemplate(dep)

	c := dep.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "/live", c.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, int32(10), c.LivenessProbe.InitialDelaySeconds, "full probe struct is preserved")
	assert.Equal(t, "/ready", c.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, "/start", c.StartupProbe.HTTPGet.Path)
}
