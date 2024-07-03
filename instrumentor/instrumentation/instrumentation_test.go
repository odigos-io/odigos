package instrumentation

// import (
// 	"encoding/json"
// 	"testing"

// 	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
// 	"github.com/odigos-io/odigos/common"
// 	"github.com/odigos-io/odigos/common/consts"
// 	"github.com/odigos-io/odigos/common/envOverwrite"
// 	"github.com/stretchr/testify/assert"
// 	appsv1 "k8s.io/api/apps/v1"
// 	v1 "k8s.io/api/core/v1"
// 	"k8s.io/apimachinery/pkg/api/resource"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// func assertNoAnnotation(t *testing.T, targetObj *appsv1.Deployment, key string) {
// 	if targetObj.GetAnnotations() != nil {
// 		_, hasAnnotation := targetObj.GetAnnotations()[key]
// 		assert.False(t, hasAnnotation)
// 	}
// }

// func assertAnnotation(t *testing.T, targetObj *appsv1.Deployment, key, value string) {
// 	assert.NotNil(t, targetObj.GetAnnotations())
// 	assert.Equal(t, value, targetObj.GetAnnotations()[key])
// }

// func assertContainerNoEnvVar(t *testing.T, podTemplate *v1.PodTemplateSpec, containerIndex int, envVarName string) {
// 	if len(podTemplate.Spec.Containers) <= containerIndex {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing container at index %d", containerIndex)
// 	}

// 	if len(podTemplate.Spec.Containers[containerIndex].Env) == 0 {
// 		return
// 	}

// 	container := podTemplate.Spec.Containers[containerIndex]
// 	for _, envVar := range container.Env {
// 		assert.NotEqual(t, envVar.Name, envVarName)
// 	}
// }

// func assertContainerWithEnvVar(t *testing.T, podTemplate *v1.PodTemplateSpec, containerIndex int, envVarName, envVarValue string) {
// 	if len(podTemplate.Spec.Containers) <= containerIndex {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing container at index %d", containerIndex)
// 	}

// 	container := podTemplate.Spec.Containers[containerIndex]
// 	assert.Contains(t, container.Env, v1.EnvVar{Name: envVarName, Value: envVarValue})
// }

// func assertContainerWithInstrumentationDevice(t *testing.T, podTemplate *v1.PodTemplateSpec, containerIndex int, instrumentationDeviceName v1.ResourceName) {

// 	if len(podTemplate.Spec.Containers) <= containerIndex {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing container at index %d", containerIndex)
// 	}

// 	container := podTemplate.Spec.Containers[containerIndex]

// 	if instrumentationDeviceName == "" {
// 		if len(container.Resources.Limits) != 0 {
// 			t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no instrumentation device in resource limits")
// 		}
// 	} else {
// 		if len(container.Resources.Limits) != 1 {
// 			t.Errorf("ApplyInstrumentationDevicesToPodTemplate() missing instrumentation device in resource limits")
// 		}
// 		_, hasInstrumentationDevice := container.Resources.Limits[instrumentationDeviceName]
// 		if !hasInstrumentationDevice {
// 			t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected instrumentation device to be set. got: %v, wanted: %v", container.Resources.Limits, instrumentationDeviceName)
// 		}
// 	}
// }

// func TestApplyInstrumentationDevicesToPodTemplate(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage: common.OtelSdkEbpfCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, v1.ResourceName("instrumentation.odigos.io/go-ebpf-community"))
// }

// func TestApplyInstrumentationDevicesToPodTemplate_MissingRuntimeDetails(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
// }

// func TestApplyInstrumentationDevicesToPodTemplate_MissingOtelSdk(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err == nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected error due to missing otel sdk")
// 	}
// }

// func TestApplyInstrumentationDevicesToPodTemplate_MultipleContainers(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{Name: "test1"},
// 				{Name: "test2"},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test1",
// 				},
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test2",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage: common.OtelSdkEbpfCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	if len(podTemplate.Spec.Containers) != 2 {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no change to number of containers")
// 	}

// 	instrumentationDeviceName := v1.ResourceName("instrumentation.odigos.io/go-ebpf-community")
// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, instrumentationDeviceName)
// 	assertContainerWithInstrumentationDevice(t, podTemplate, 1, instrumentationDeviceName)
// }

// func TestApplyInstrumentationDevicesToPodTemplate_MultipleHeterogeneousContainers(t *testing.T) {

// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{Name: "test1"},
// 				{Name: "test2"},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test1",
// 				},
// 				{
// 					Language:      common.JavaProgrammingLanguage,
// 					ContainerName: "test2",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage:   common.OtelSdkEbpfCommunity,
// 		common.JavaProgrammingLanguage: common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	if len(podTemplate.Spec.Containers) != 2 {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no change to number of containers")
// 	}

// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, v1.ResourceName("instrumentation.odigos.io/go-ebpf-community"))
// 	assertContainerWithInstrumentationDevice(t, podTemplate, 1, v1.ResourceName("instrumentation.odigos.io/java-native-community"))
// }

// func TestApplyInstrumentationDevicesToPodTemplate_MultiplePartialContainers(t *testing.T) {

// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{Name: "test1"},
// 				{Name: "test2"},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.JavaProgrammingLanguage,
// 					ContainerName: "test2",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage:   common.OtelSdkEbpfCommunity,
// 		common.JavaProgrammingLanguage: common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	if len(podTemplate.Spec.Containers) != 2 {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected no change to number of containers")
// 	}

// 	// container 0 should not be modified because it is not in the runtime details
// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")

// 	assertContainerWithInstrumentationDevice(t, podTemplate, 1, v1.ResourceName("instrumentation.odigos.io/java-native-community"))
// }

// func TestApplyInstrumentationDevicesToPodTemplate_AppendExistingLimits(t *testing.T) {

// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 					Resources: v1.ResourceRequirements{
// 						Limits: map[v1.ResourceName]resource.Quantity{
// 							"foo": resource.MustParse("123"),
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage: common.OtelSdkEbpfCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	container := podTemplate.Spec.Containers[0]

// 	if len(container.Resources.Limits) != 2 {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected 2 resource limits")
// 	}

// 	if container.Resources.Limits["foo"] != resource.MustParse("123") {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected existing resource limit to be preserved")
// 	}

// 	if container.Resources.Limits["instrumentation.odigos.io/go-ebpf-community"] != resource.MustParse("1") {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected instrumentation device to be added")
// 	}
// }

// func TestApplyInstrumentationDevicesToPodTemplate_RemoveExistingLimits(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 					Resources: v1.ResourceRequirements{
// 						Limits: map[v1.ResourceName]resource.Quantity{
// 							"instrumentation.odigos.io/go-ebpf-community": resource.MustParse("1"),
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage: common.OtelSdkEbpfEnterprise,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	container := podTemplate.Spec.Containers[0]

// 	if len(container.Resources.Limits) != 1 {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected 1 resource limits")
// 	}

// 	if _, ok := container.Resources.Limits["instrumentation.odigos.io/go-ebpf-community"]; ok {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected to remove old existing resource limit for community ")
// 	}

// 	if container.Resources.Limits["instrumentation.odigos.io/go-ebpf-enterprise"] != resource.MustParse("1") {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() expected instrumentation device to be added")
// 	}
// }

// func TestRevert(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 					Env: []v1.EnvVar{
// 						{
// 							Name:  "PYTHONPATH",
// 							Value: "/very/important/path",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.PythonProgrammingLanguage,
// 					ContainerName: "test",
// 				},
// 			},
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.PythonProgrammingLanguage: common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	// make sure the env var is appended
// 	val, ok := envOverwrite.ValToAppend("PYTHONPATH", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)

// 	want := "/very/important/path:" + val
// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)

// 	// The original value of the env var should be stored in an annotation
// 	a, _ := json.Marshal(map[string]map[string]string{
// 		"test": {
// 			"PYTHONPATH": "/very/important/path",
// 		},
// 	})
// 	want = string(a)
// 	assertAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation, want)

// 	RevertInstrumentationDevices(podTemplate, deployment)
// 	// The env var should be reverted to its original value
// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", "/very/important/path")

// 	if len(podTemplate.Spec.Containers) != 1 {
// 		t.Errorf("Revert() expected no change to number of containers")
// 	}

// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
// 	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
// }

// func TestRevert_ExistingResources(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "test",
// 					Resources: v1.ResourceRequirements{
// 						Limits: map[v1.ResourceName]resource.Quantity{
// 							"foo": resource.MustParse("123"),
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage: common.OtelSdkEbpfCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	RevertInstrumentationDevices(podTemplate, deployment)

// 	if len(podTemplate.Spec.Containers) != 1 {
// 		t.Errorf("Revert() expected no change to number of containers")
// 	}

// 	if len(podTemplate.Spec.Containers[0].Resources.Limits) != 1 {
// 		t.Errorf("Revert() expected no change to number of resource limits")
// 	}

// 	if podTemplate.Spec.Containers[0].Resources.Limits["foo"] != resource.MustParse("123") {
// 		t.Errorf("Revert() expected existing resource limit to be preserved")
// 	}
// }

// func TestRevert_MultipleContainers(t *testing.T) {

// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{Name: "test1"},
// 				{Name: "test2"},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test1",
// 				},
// 				{
// 					Language:      common.GoProgrammingLanguage,
// 					ContainerName: "test2",
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.GoProgrammingLanguage: common.OtelSdkEbpfCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	RevertInstrumentationDevices(podTemplate, deployment)

// 	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
// 	assertContainerWithInstrumentationDevice(t, podTemplate, 1, "")
// }

// func TestEnvVarAppendMultipleContainers(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "pythonContainer",
// 					Env: []v1.EnvVar{
// 						{
// 							Name:  "PYTHONPATH",
// 							Value: "/very/important/path",
// 						},
// 					},
// 				},
// 				{
// 					Name: "nodeContainer",
// 					Env: []v1.EnvVar{
// 						{
// 							Name:  "NODE_OPTIONS",
// 							Value: "--max-old-space-size=8192",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.PythonProgrammingLanguage,
// 					ContainerName: "pythonContainer",
// 				},
// 				{
// 					Language:      common.JavascriptProgrammingLanguage,
// 					ContainerName: "nodeContainer",
// 				},
// 			},
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
// 		common.PythonProgrammingLanguage:     common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	pythonpathVal, ok := envOverwrite.ValToAppend("PYTHONPATH", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)
// 	want := "/very/important/path:" + pythonpathVal
// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)

// 	nodeOptionsVal, ok := envOverwrite.ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)
// 	want = "--max-old-space-size=8192 " + nodeOptionsVal
// 	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", want)

// 	// The original value of the env var should be stored in an annotation
// 	a, _ := json.Marshal(map[string]map[string]string{
// 		"pythonContainer": {
// 			"PYTHONPATH": "/very/important/path",
// 		},
// 		"nodeContainer": {
// 			"NODE_OPTIONS": "--max-old-space-size=8192",
// 		},
// 	})
// 	want = string(a)
// 	assertAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation, want)

// 	RevertInstrumentationDevices(podTemplate, deployment)

// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", "/very/important/path")
// 	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", "--max-old-space-size=8192")
// 	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
// }

// func TestEnvVarFromRuntimeDetails(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{Name: "pythonContainer"},
// 				{Name: "nodeContainer"},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.PythonProgrammingLanguage,
// 					ContainerName: "pythonContainer",
// 					EnvVars: []odigosv1.EnvVar{
// 						{
// 							Name:  "PYTHONPATH",
// 							Value: "/very/important/path",
// 						},
// 					},
// 				},
// 				{
// 					Language:      common.JavascriptProgrammingLanguage,
// 					ContainerName: "nodeContainer",
// 					EnvVars: []odigosv1.EnvVar{
// 						{
// 							Name:  "NODE_OPTIONS",
// 							Value: "--max-old-space-size=8192",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
// 		common.PythonProgrammingLanguage:     common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	pythonpathVal, ok := envOverwrite.ValToAppend("PYTHONPATH", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)
// 	want := "/very/important/path:" + pythonpathVal
// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)

// 	nodeOptionsVal, ok := envOverwrite.ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)
// 	want = "--max-old-space-size=8192 " + nodeOptionsVal
// 	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", want)

// 	// The env vars originated from the runtime details should not be stored in the annotation
// 	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)

// 	RevertInstrumentationDevices(podTemplate, deployment)
// 	// After reverting, the env vars should not be present in the container since they originated from the runtime details
// 	// and were not present in the original pod template
// 	assertContainerNoEnvVar(t, podTemplate, 0, "PYTHONPATH")
// 	assertContainerNoEnvVar(t, podTemplate, 1, "NODE_OPTIONS")
// }

// func TestEnvVarAppendFromSpecAndRuntimeDetails(t *testing.T) {
// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "pythonContainer",
// 					Env: []v1.EnvVar{
// 						{
// 							Name:  "PYTHONPATH",
// 							Value: "/very/important/path/template",
// 						},
// 					},
// 				},
// 				{
// 					Name: "nodeContainer",
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.PythonProgrammingLanguage,
// 					ContainerName: "pythonContainer",
// 					EnvVars: []odigosv1.EnvVar{
// 						{
// 							Name:  "PYTHONPATH",
// 							Value: "/very/important/path/runtime",
// 						},
// 					},
// 				},
// 				{
// 					Language:      common.JavascriptProgrammingLanguage,
// 					ContainerName: "nodeContainer",
// 					EnvVars: []odigosv1.EnvVar{
// 						{
// 							Name:  "NODE_OPTIONS",
// 							Value: "--max-old-space-size=8192-runtime",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
// 		common.PythonProgrammingLanguage:     common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	// If env vars are present in both the template and runtime details, the template value should be used (pythonContainer in this case)
// 	pythonpathVal, ok := envOverwrite.ValToAppend("PYTHONPATH", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)
// 	want := "/very/important/path/template:" + pythonpathVal
// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", want)

// 	// The env var from the runtime details should be used for nodeContainer since it is not present in the template
// 	nodeOptionsVal, ok := envOverwrite.ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
// 	assert.True(t, ok)
// 	want = "--max-old-space-size=8192-runtime " + nodeOptionsVal
// 	assertContainerWithEnvVar(t, podTemplate, 1, "NODE_OPTIONS", want)
// 	// The original value of the env var should be stored in an annotation
// 	a, _ := json.Marshal(map[string]map[string]string{
// 		"pythonContainer": {
// 			"PYTHONPATH": "/very/important/path/template",
// 		},
// 	})
// 	want = string(a)
// 	assertAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation, want)

// 	RevertInstrumentationDevices(podTemplate, deployment)
// 	// After reverting, make sure we are back to the original state
// 	assertContainerWithEnvVar(t, podTemplate, 0, "PYTHONPATH", "/very/important/path/template")
// 	assertContainerNoEnvVar(t, podTemplate, 1, "NODE_OPTIONS")
// 	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
// }

// func TestMoveBetweenSDKsWithUserValue(t *testing.T) {
// 	jsOptionsValEbpf, _ := envOverwrite.ValToAppend("NODE_OPTIONS", common.OtelSdkEbpfEnterprise)
// 	jsOptionsValNative, _ := envOverwrite.ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
// 	userDefinedVal := "--max-old-space-size=8192-runtime"

// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "nodeContainer",
// 					Env: []v1.EnvVar{
// 						{
// 							Name:  "NODE_OPTIONS",
// 							Value: userDefinedVal + " " + jsOptionsValEbpf,
// 						},
// 					},
// 				},
// 				{
// 					Name: "nodeContainer",
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.JavascriptProgrammingLanguage,
// 					ContainerName: "nodeContainer",
// 					EnvVars: []odigosv1.EnvVar{
// 						{
// 							Name:  "NODE_OPTIONS",
// 							Value: userDefinedVal,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	want := userDefinedVal + " " + jsOptionsValNative
// 	assertContainerWithEnvVar(t, podTemplate, 0, "NODE_OPTIONS", want)

// 	RevertInstrumentationDevices(podTemplate, deployment)
// 	assertNoAnnotation(t, deployment, consts.ManifestEnvOriginalValAnnotation)
// }

// func TestMoveBetweenSDKsWithoutUserValue(t *testing.T) {
// 	jsOptionsValEbpf, _ := envOverwrite.ValToAppend("NODE_OPTIONS", common.OtelSdkEbpfEnterprise)

// 	podTemplate := &v1.PodTemplateSpec{
// 		Spec: v1.PodSpec{
// 			Containers: []v1.Container{
// 				{
// 					Name: "nodeContainer",
// 				},
// 			},
// 		},
// 	}

// 	runtimeDetails := &odigosv1.InstrumentedApplication{
// 		Spec: odigosv1.InstrumentedApplicationSpec{
// 			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
// 				{
// 					Language:      common.JavascriptProgrammingLanguage,
// 					ContainerName: "nodeContainer",
// 					EnvVars: []odigosv1.EnvVar{
// 						{
// 							Name:  "NODE_OPTIONS",
// 							Value: jsOptionsValEbpf,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test",
// 			Namespace: "test",
// 		},
// 	}

// 	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
// 		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
// 	}

// 	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks, deployment)
// 	if err != nil {
// 		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
// 	}

// 	assertContainerNoEnvVar(t, podTemplate, 0, "NODE_OPTIONS")
// }
