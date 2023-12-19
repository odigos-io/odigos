package instrumentation

import (
	"testing"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

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
			Languages: []common.LanguageByContainer{
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

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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
			Languages: []common.LanguageByContainer{},
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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
			Languages: []common.LanguageByContainer{
				{
					Language:      common.GoProgrammingLanguage,
					ContainerName: "test",
				},
			},
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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
			Languages: []common.LanguageByContainer{
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

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage: {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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
			Languages: []common.LanguageByContainer{
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

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage:   {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.JavaProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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
			Languages: []common.LanguageByContainer{
				{
					Language:      common.JavaProgrammingLanguage,
					ContainerName: "test2",
				},
			},
		},
	}

	defaultSdks := map[common.ProgrammingLanguage]common.OtelSdk{
		common.GoProgrammingLanguage:   {SdkType: common.EbpfOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
		common.JavaProgrammingLanguage: {SdkType: common.NativeOtelSdkType, SdkTier: common.CommunityOtelSdkTier},
	}

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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
			Languages: []common.LanguageByContainer{
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

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
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

func TestRevert(t *testing.T) {
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
			Languages: []common.LanguageByContainer{
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

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	Revert(podTemplate)

	if len(podTemplate.Spec.Containers) != 1 {
		t.Errorf("Revert() expected no change to number of containers")
	}

	assertContainerWithInstrumentationDevice(t, podTemplate, 0, "")
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
			Languages: []common.LanguageByContainer{
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

	err := ApplyInstrumentationDevicesToPodTemplate(podTemplate, runtimeDetails, defaultSdks)
	if err != nil {
		t.Errorf("ApplyInstrumentationDevicesToPodTemplate() error = %v", err)
	}

	Revert(podTemplate)

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
