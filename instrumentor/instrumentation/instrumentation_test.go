package instrumentation

import (
	"encoding/json"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func assertNoAnnotation(t *testing.T, targetObj *appsv1.Deployment, key string) {
	if targetObj.GetAnnotations() != nil {
		_, hasAnnotation := targetObj.GetAnnotations()[key]
		assert.False(t, hasAnnotation)
	}
}

func assertAnnotation(t *testing.T, targetObj *appsv1.Deployment, key, value string) {
	assert.NotNil(t, targetObj.GetAnnotations())
	assert.Equal(t, value, targetObj.GetAnnotations()[key])
}

func assertContainerNoEnvVar(t *testing.T, podTemplate *v1.PodTemplateSpec, containerIndex int, envVarName string) {
	if len(podTemplate.Spec.Containers) <= containerIndex {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing container at index %d", containerIndex)
	}

	container := podTemplate.Spec.Containers[containerIndex]
	for _, envVar := range container.Env {
		assert.NotEqual(t, envVar.Name, envVarName)
	}
}

func assertContainerWithEnvVar(t *testing.T, podTemplate *v1.PodTemplateSpec, containerIndex int, envVarName, envVarValue string) {
	if len(podTemplate.Spec.Containers) <= containerIndex {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing container at index %d", containerIndex)
	}

	container := podTemplate.Spec.Containers[containerIndex]
	assert.Contains(t, container.Env, v1.EnvVar{Name: envVarName, Value: envVarValue})
}

func assertContainerWithInstrumentationDevice(t *testing.T, podTemplate *v1.PodTemplateSpec, containerIndex int, instrumentationDeviceName v1.ResourceName) {

	if len(podTemplate.Spec.Containers) <= containerIndex {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing container at index %d", containerIndex)
	}

	container := podTemplate.Spec.Containers[containerIndex]

	if instrumentationDeviceName == "" {
		if len(container.Resources.Limits) != 0 {
			t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no instrumentation device in resource limits")
		}
	} else {
		if len(container.Resources.Limits) != 1 {
			t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing instrumentation device in resource limits")
		}
		_, hasInstrumentationDevice := container.Resources.Limits[instrumentationDeviceName]
		if !hasInstrumentationDevice {
			t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected instrumentation device to be set. got: %v, wanted: %v", container.Resources.Limits, instrumentationDeviceName)
		}
	}
}

func TestApplyInstrumentationDevicesToPodTemplate(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	assertContainerWithInstrumentationDevice(t, podTemplate, 0, v1.ResourceName("instrumentation.odigos.io/go-ebpf-community"))
}

func TestApplyInstrumentationDevicesToPodTemplate_MissingRuntimeDetails(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
}

func TestApplyInstrumentationDevicesToPodTemplate_MissingOtelSdk(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err == nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected error due to missing otel sdk")
	}
}

func TestApplyInstrumentationDevicesToPodTemplate_MultipleContainers(t *testing.T) {

	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test1",
				},
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test2",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	if len(podTemplate.Spec.Containers) != 2 {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no change to number of containers")
	}

	instrumentationDeviceName := v1.ResourceName("instrumentation.odigos.io/go-ebpf-community")
	assertContainerWithInstrumentationDevice(t, podTemplate, 0, instrumentationDeviceName)
	assertContainerWithInstrumentationDevice(t, podTemplate, 1, instrumentationDeviceName)
}

func TestApplyInstrumentationDevicesToPodTemplate_MultipleHeterogeneousContainers(t *testing.T) {

	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test1",
				},
				{
					Language:      common.JavaProgrammingLanguage,
					ContainerName: "test2",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage:   {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.JavaProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	if len(podTemplate.Spec.Containers) != 2 {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no change to number of containers")
	}

	assertContainerWithInstrumentationDevice(t, podTemplate, 0, v1.ResourceName("instrumentation.odigos.io/go-ebpf-community"))
	assertContainerWithInstrumentationDevice(t, podTemplate, 1, v1.ResourceName("instrumentation.odigos.io/java-native-community"))
}

func TestApplyInstrumentationDevicesToPodTemplate_MultiplePartialContainers(t *testing.T) {

	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.JavaProgrammingLanguage,
					ContainerName: "test2",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage:   {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.JavaProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	if len(podTemplate.Spec.Containers) != 2 {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no change to number of containers")
	}

	// container 0 should not be modified because it is not in the runtime details
	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")

	assertContainerWithInstrumentationDevice(t, podTemplate, 1, v1.ResourceName("instrumentation.odigos.io/java-native-community"))
}

func TestApplyInstrumentationDevicesToPodTemplate_AppendExistingLimits(t *testing.T) {

	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
					Resources: v1.ResourceRequirements{
						Limits: map[v1.ResourceName]resource.Quantity{
							"foo": resource.MustParse("123"),
						},
					},
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	container := podTemplate.Spec.Containers[0]

	if len(container.Resources.Limits) != 2 {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected 2 resource limits")
	}

	if container.Resources.Limits["foo"] != resource.MustParse("123") {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected existing resource limit to be preserved")
	}

	if container.Resources.Limits["instrumentation.odigos.io/go-ebpf-community"] != resource.MustParse("1") {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected instrumentation device to be added")
	}
}

func TestApplyInstrumentationDevicesToPodTemplate_RemoveExistingLimits(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
					Resources: v1.ResourceRequirements{
						Limits: map[v1.ResourceName]resource.Quantity{
							"instrumentation.odigos.io/go-ebpf-community": resource.MustParse("1"),
						},
					},
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.EnterpriseOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	container := podTemplate.Spec.Containers[0]

	if len(container.Resources.Limits) != 1 {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected 1 resource limits")
	}

	if _, ok := container.Resources.Limits["instrumentation.odigos.io/go-ebpf-community"]; ok {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected to remove old existing resource limit for community ")
	}

	if container.Resources.Limits["instrumentation.odigos.io/go-ebpf-enterprise"] != resource.MustParse("1") {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected instrumentation device to be added")
	}
}

func TestRevert(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
					Env: []v1.EnvVar{
						{
							Name:  "PYTHONPATH",
							Value: "/very/important/path",
						},
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	// make sure the env var is appended
	want := "/very/important/path:" + envOverwrite.EnvValues["PYTHONPATH"].Value
	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)

	// The original value of the env var should be stored in an annotation
	a, _ := json.Marshal(map[string]map[string]string{
		"test": {
			"PYTHONPATH": "/very/important/path",
		},
	})
	want = string(a)
	assertAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation, want)

	Revert(podTemplate, deployment)
	// The env var should be reverted to its original value
	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", "/very/important/path")

	if len(podTemplate.Spec.Containers) != 1 {
		t.Errorf("Revert() expected no change to number of containers")
	}

	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
}

func TestRevert_ExistingResources(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test",
					Resources: v1.ResourceRequirements{
						Limits: map[v1.ResourceName]resource.Quantity{
							"foo": resource.MustParse("123"),
						},
					},
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	Revert(podTemplate, deployment)

	if len(podTemplate.Spec.Containers) != 1 {
		t.Errorf("Revert() expected no change to number of containers")
	}

	if len(podTemplate.Spec.Containers[0].Resources.Limits) != 1 {
		t.Errorf("Revert() expected no change to number of resource limits")
	}

	if podTemplate.Spec.Containers[0].Resources.Limits["foo"] != resource.MustParse("123") {
		t.Errorf("Revert() expected existing resource limit to be preserved")
	}
}

func TestRevert_MultipleContainers(t *testing.T) {

	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test1",
				},
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test2",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	Revert(podTemplate, deployment)

	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
	assertContainerWithInstrumentationDevice(t, podTemplate, 1, "")
}

func TestEnvVarAppendMultipleContainers(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "pythonContainer",
					Env: []v1.EnvVar{
						{
							Name:  "PYTHONPATH",
							Value: "/very/important/path",
						},
					},
				},
				{
					Name: "nodeContainer",
					Env: []v1.EnvVar{
						{
							Name:  "NODE_OPTIONS",
							Value: "--max-old-space-size=8192",
						},
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.PythonProgrammingLanguage,
					ContainerName: "pythonContainer",
				},
				{
					Language:      common.JavascriptProgrammingLanguage,
					ContainerName: "nodeContainer",
				},
			},
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavascriptProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.PythonProgrammingLanguage:     {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	want := "/very/important/path:" + envOverwrite.EnvValues["PYTHONPATH"].Value
	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)
	want = "--max-old-space-size=8192 " + envOverwrite.EnvValues["NODE_OPTIONS"].Value
	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", want)

	// The original value of the env var should be stored in an annotation
	a, _ := json.Marshal(map[string]map[string]string{
		"pythonContainer": {
			"PYTHONPATH": "/very/important/path",
		},
		"nodeContainer": {
			"NODE_OPTIONS": "--max-old-space-size=8192",
		},
	})
	want = string(a)
	assertAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation, want)

	Revert(podTemplate, deployment)

	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", "/very/important/path")
	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", "--max-old-space-size=8192")
	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
}

func TestEnvVarFromRuntimeDetails(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "pythonContainer"},
				{Name: "nodeContainer"},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.PythonProgrammingLanguage,
					ContainerName: "pythonContainer",
					EnvVars: []odigosv1.EnvVar{
						{
							Name:  "PYTHONPATH",
							Value: "/very/important/path",
						},
					},
				},
				{
					Language:      common.JavascriptProgrammingLanguage,
					ContainerName: "nodeContainer",
					EnvVars: []odigosv1.EnvVar{
						{
							Name:  "NODE_OPTIONS",
							Value: "--max-old-space-size=8192",
						},
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavascriptProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.PythonProgrammingLanguage:     {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	want := "/very/important/path:" + envOverwrite.EnvValues["PYTHONPATH"].Value
	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)
	want = "--max-old-space-size=8192 " + envOverwrite.EnvValues["NODE_OPTIONS"].Value
	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", want)

	// The env vars originated from the runtime details should not be stored in the annotation
	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)

	Revert(podTemplate, deployment)
	// After reverting, the env vars should not be present in the container since they originated from the runtime details
	// and were not present in the original pod template
	assertContainerNoEnvVar(t, podTemplate, 0, "PYTHONPATH")
	assertContainerNoEnvVar(t, podTemplate, 1, "NODE_OPTIONS")
}

func TestEnvVarAppendFromSpecAndRuntimeDetails(t *testing.T) {
	podTemplate := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "pythonContainer",
					Env: []v1.EnvVar{
						{
							Name:  "PYTHONPATH",
							Value: "/very/important/path/template",
						},
					},
				},
				{
					Name: "nodeContainer",
				},
			},
		},
	}

	runtimeDetails := &odigosv1.InstrumentedApplication{
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					Language:      common.PythonProgrammingLanguage,
					ContainerName: "pythonContainer",
					EnvVars: []odigosv1.EnvVar{
						{
							Name:  "PYTHONPATH",
							Value: "/very/important/path/runtime",
						},
					},
				},
				{
					Language:      common.JavascriptProgrammingLanguage,
					ContainerName: "nodeContainer",
					EnvVars: []odigosv1.EnvVar{
						{
							Name:  "NODE_OPTIONS",
							Value: "--max-old-space-size=8192-runtime",
						},
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavascriptProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.PythonProgrammingLanguage:     {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	// If env vars are present in both the template and runtime details, the template value should be used (pythonContainer in this case)
	want := "/very/important/path/template:" + envOverwrite.EnvValues["PYTHONPATH"].Value
	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)
	// The env var from the runtime details should be used for nodeContainer since it is not present in the template
	want = "--max-old-space-size=8192-runtime " + envOverwrite.EnvValues["NODE_OPTIONS"].Value
	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", want)
	// The original value of the env var should be stored in an annotation
	a, _ := json.Marshal(map[string]map[string]string{
		"pythonContainer": {
			"PYTHONPATH": "/very/important/path/template",
		},
	})
	want = string(a)
	assertAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation, want)

	Revert(podTemplate, deployment)
	// After reverting, make sure we are back to the original state
	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", "/very/important/path/template")
	assertContainerNoEnvVar(t, podTemplate, 1, "NODE_OPTIONS")
	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
}
