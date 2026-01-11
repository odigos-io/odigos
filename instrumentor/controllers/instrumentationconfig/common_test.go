package instrumentationconfig

import (
	"slices"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateInstrumentationConfigForWorkload_SingleLanguage(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if ic.Spec.SdkConfigs[0].Language != common.JavascriptProgrammingLanguage {
		t.Errorf("Expected language %s, got %s", common.JavascriptProgrammingLanguage, ic.Spec.SdkConfigs[0].Language)
	}
}

func TestUpdateInstrumentationConfigForWorkload_MultipleLanguages(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container-1",
				},
				{
					ContainerName: "test-container-2",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container-1",
					Language:      common.JavascriptProgrammingLanguage,
				},
				{
					ContainerName: "test-container-2",
					Language:      common.PythonProgrammingLanguage,
				},
			},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 2 {
		t.Errorf("Expected 2 sdk configs, got %d", len(ic.Spec.SdkConfigs))
	}

	gotLanguages := []common.ProgrammingLanguage{}
	for i := range ic.Spec.SdkConfigs {
		gotLanguages = append(gotLanguages, ic.Spec.SdkConfigs[i].Language)
	}

	slices.Sort(gotLanguages)
	if slices.Compare(gotLanguages, []common.ProgrammingLanguage{
		common.JavascriptProgrammingLanguage,
		common.PythonProgrammingLanguage,
	}) != 0 {
		t.Errorf("Expected Python and Javascript as languages, got %s", gotLanguages)
	}
}

func TestUpdateInstrumentationConfigForWorkload_IgnoreUnknownLanguage(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container-1",
				},
				{
					ContainerName: "test-container-2",
				},
				{
					ContainerName: "test-container-3",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container-1",
					Language:      common.JavascriptProgrammingLanguage,
				},
				{
					ContainerName: "test-container-2",
					Language:      common.UnknownProgrammingLanguage,
				},
			},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if ic.Spec.SdkConfigs[0].Language != common.JavascriptProgrammingLanguage {
		t.Errorf("Expected language %s, got %s", common.JavascriptProgrammingLanguage, ic.Spec.SdkConfigs[0].Language)
	}
}

func TestUpdateInstrumentationConfigForWorkload_NoLanguages(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 0 {
		t.Errorf("Expected 0 sdk configs, got %d", len(ic.Spec.SdkConfigs))
	}
}

func TestUpdateInstrumentationConfigForWorkload_SameLanguageMultipleContainers(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container-1",
				},
				{
					ContainerName: "test-container-2",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container-1",
					Language:      common.JavascriptProgrammingLanguage,
				},
				{
					ContainerName: "test-container-2",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if ic.Spec.SdkConfigs[0].Language != common.JavascriptProgrammingLanguage {
		t.Errorf("Expected language %s, got %s", common.JavascriptProgrammingLanguage, ic.Spec.SdkConfigs[0].Language)
	}
}

func TestUpdateInstrumentationConfigForWorkload_SingleMatchingRule(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes:           &[]string{"application/json"},
							MaxPayloadLength:    Int64Ptr(1234),
							DropPartialPayloads: BoolPtr(true),
						},
					},
				},
			},
		},
	}
	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes))
	}
	if (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[0] != "application/json" {
		t.Errorf("Expected mime type %s, got %s", "application/json", (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[0])
	}
	if *ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MaxPayloadLength != 1234 {
		t.Errorf("Expected max payload length %d, got %d", 1234, ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MaxPayloadLength)
	}
	if *ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.DropPartialPayloads != true {
		t.Errorf("Expected drop partial payloads %t, got %t", true, *ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.DropPartialPayloads)
	}
}

func TestUpdateInstrumentationConfigForWorkload_InWorkloadList(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					Workloads: &[]k8sconsts.PodWorkload{
						{
							Name:      "test",
							Kind:      k8sconsts.WorkloadKindDeployment,
							Namespace: "testns",
						},
					},
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes))
	}
}

func TestUpdateInstrumentationConfigForWorkload_NotInWorkloadList(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					Workloads: &[]k8sconsts.PodWorkload{
						{
							Name:      "someotherdeployment",
							Kind:      k8sconsts.WorkloadKindDeployment,
							Namespace: "testns",
						},
					},
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 0 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	// rule should be ignored since "test" deployment is not in the workload list
	if ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest != nil {
		t.Errorf("Expected nil, got %v", ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest)
	}
}

func TestUpdateInstrumentationConfigForWorkload_DisabledRule(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					Disabled: true,
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	// rule should be ignored since it is disabled
	if ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest != nil {
		t.Errorf("Expected nil, got %v", ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest)
	}
}

func TestUpdateInstrumentationConfigForWorkload_MultipleDefaultRules(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes:           &[]string{"application/json", "application/text"},
							MaxPayloadLength:    Int64Ptr(1111),
							DropPartialPayloads: BoolPtr(true),
						},
					},
				},
			},
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes:           &[]string{"application/xml", "application/json"},
							MaxPayloadLength:    Int64Ptr(2222),
							DropPartialPayloads: BoolPtr(false),
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}

	// mime types should merge
	mimeTypes := ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes
	if len(*mimeTypes) != 3 {
		t.Errorf("Expected 2 mime types, got %d", len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes))
	}
	expectedMimeTypes := map[string]bool{
		"application/json": true,
		"application/xml":  true,
		"application/text": true,
	}
	for _, mt := range *mimeTypes {
		if !expectedMimeTypes[mt] {
			t.Errorf("Unexpected mime type %s", mt)
		}
	}
	// Ensure all expected mime types are present
	for expected := range expectedMimeTypes {
		found := false
		for _, actual := range *mimeTypes {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected mime type %s", expected)
		}
	}

	// smallest max payload length should be selected
	if *ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MaxPayloadLength != 1111 {
		t.Errorf("Expected max payload length %d, got %d", 1111, ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MaxPayloadLength)
	}
	// one of the rules has drop partial payloads set to true, so it should be true
	if *ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.DropPartialPayloads != true {
		t.Errorf("Expected drop partial payloads %t, got %t", true, *ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.DropPartialPayloads)
	}
}

func TestUpdateInstrumentationConfigForWorkload_RuleForLibrary(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					InstrumentationLibraries: &[]odigosv1.InstrumentationLibraryGlobalId{
						{
							Name:     "test-library",
							Language: common.JavascriptProgrammingLanguage,
						},
					},
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest != nil {
		t.Errorf("Expected nil, got %v", ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest)
	}
	if len(ic.Spec.SdkConfigs[0].InstrumentationLibraryConfigs) != 1 {
		t.Errorf("Expected 1 library, got %d", len(ic.Spec.SdkConfigs[0].InstrumentationLibraryConfigs))
	}
	if len(*ic.Spec.SdkConfigs[0].InstrumentationLibraryConfigs[0].PayloadCollection.HttpRequest.MimeTypes) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(*ic.Spec.SdkConfigs[0].InstrumentationLibraryConfigs[0].PayloadCollection.HttpRequest.MimeTypes))
	}
}

func TestUpdateInstrumentationConfigForWorkload_LibraryRuleOtherLanguage(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.JavascriptProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					Disabled: true,
					InstrumentationLibraries: &[]odigosv1.InstrumentationLibraryGlobalId{
						{
							Name:     "test-library",
							Language: common.PythonProgrammingLanguage, // Notice, the library is for python and sdk language is javascript
						},
					},
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest: &instrumentationrules.HttpPayloadCollection{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	if ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest != nil {
		t.Errorf("Expected nil, got %v", ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest)
	}
	if len(ic.Spec.SdkConfigs[0].InstrumentationLibraryConfigs) != 0 { // the library specified is for different language
		t.Errorf("Expected 0 libraries, got %d", len(ic.Spec.SdkConfigs[0].InstrumentationLibraryConfigs))
	}
}

func TestUpdateInstrumentationConfigForWorkload_RuleFoOverrideContainer(t *testing.T) {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1.ContainerOverride{
				{
					ContainerName: "test-container",
					RuntimeInfo: &odigosv1.RuntimeDetailsByContainer{
						ContainerName: "test-container",
						Language:      common.GoProgrammingLanguage,
					},
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container",
					Language:      common.NginxProgrammingLanguage,
				},
			},
		},
	}

	rules := &odigosv1.InstrumentationRuleList{
		Items: []odigosv1.InstrumentationRule{
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					PayloadCollection: &instrumentationrules.PayloadCollection{
						HttpRequest:  &instrumentationrules.HttpPayloadCollection{},
						HttpResponse: &instrumentationrules.HttpPayloadCollection{},
						Messaging:    &instrumentationrules.MessagingPayloadCollection{},
						DbQuery:      &instrumentationrules.DbQueryPayloadCollection{},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, rules, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}
	sdkConfig := ic.Spec.SdkConfigs[0]
	// make sure the override got into account
	if sdkConfig.Language != common.GoProgrammingLanguage {
		t.Errorf("Expected to get golang language for sdk config, got %s", ic.Spec.SdkConfigs[0].Language)
	}
	if sdkConfig.DefaultPayloadCollection == nil {
		t.Errorf("expected to have non nil default payload collection")
	}
	if sdkConfig.DefaultPayloadCollection.DbQuery == nil {
		t.Error("expected to have non nil db query config")
	}
	if sdkConfig.DefaultPayloadCollection.HttpRequest == nil {
		t.Errorf("expected to have non nil http request config")
	}
	if sdkConfig.DefaultPayloadCollection.HttpResponse == nil {
		t.Errorf("expected to have non nil http response config")
	}
	if sdkConfig.DefaultPayloadCollection.Messaging == nil {
		t.Errorf("expected to have non nil messaging config")
	}
}

func TestMergeHttpPayloadCollectionRules(t *testing.T) {
	res := mergeHttpPayloadCollectionRules(&instrumentationrules.HttpPayloadCollection{
		MimeTypes:        &[]string{"application/json"},
		MaxPayloadLength: Int64Ptr(1234),
	}, &instrumentationrules.HttpPayloadCollection{
		MimeTypes:           &[]string{"application/xml"},
		DropPartialPayloads: BoolPtr(false),
	})
	if len(*res.MimeTypes) != 2 {
		t.Errorf("Expected 2 mime types, got %d", len(*res.MimeTypes))
	}
	// Test MimeTypes without considering the order
	expectedMimeTypes := map[string]bool{
		"application/json": true,
		"application/xml":  true,
	}
	for _, mt := range *res.MimeTypes {
		if !expectedMimeTypes[mt] {
			t.Errorf("Unexpected mime type %s", mt)
		}
	}
	// Ensure all expected mime types are present
	for expected := range expectedMimeTypes {
		found := false
		for _, actual := range *res.MimeTypes {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected mime type %s", expected)
		}
	}

	if *res.MaxPayloadLength != 1234 {
		t.Errorf("Expected max payload length %d, got %d", 1234, *res.MaxPayloadLength)
	}
	if *res.DropPartialPayloads != false {
		t.Errorf("Expected drop partial payloads %t, got %t", false, *res.DropPartialPayloads)
	}
}

func TestMergeHttpPayloadCollectionRules_BothNil(t *testing.T) {
	res := mergeHttpPayloadCollectionRules(nil, nil)
	if res != nil {
		t.Errorf("Expected nil, got %v", res)
	}
}

func TestMergeHttpPayloadCollectionRules_FirstNil(t *testing.T) {
	res := mergeHttpPayloadCollectionRules(nil, &instrumentationrules.HttpPayloadCollection{
		MimeTypes:           &[]string{"application/xml"},
		DropPartialPayloads: BoolPtr(false),
	})
	if len(*res.MimeTypes) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(*res.MimeTypes))
	}
	if (*res.MimeTypes)[0] != "application/xml" {
		t.Errorf("Expected mime type %s, got %s", "application/xml", (*res.MimeTypes)[0])
	}
	if res.MaxPayloadLength != nil {
		t.Errorf("Expected nil, got %v", res.MaxPayloadLength)
	}
	if *res.DropPartialPayloads != false {
		t.Errorf("Expected drop partial payloads %t, got %t", false, *res.DropPartialPayloads)
	}
}

func TestMergeHttpPayloadCollectionRules_SecondNil(t *testing.T) {
	res := mergeHttpPayloadCollectionRules(&instrumentationrules.HttpPayloadCollection{
		MaxPayloadLength: Int64Ptr(1234),
	}, nil)
	if res.MimeTypes != nil {
		t.Errorf("Expected nil, got %v", res.MimeTypes)
	}
	if *res.MaxPayloadLength != 1234 {
		t.Errorf("Expected max payload length %d, got %d", 1234, *res.MaxPayloadLength)
	}
	if res.DropPartialPayloads != nil {
		t.Errorf("Expected nil, got %v", res.DropPartialPayloads)
	}
}

func BoolPtr(b bool) *bool {
	return &b
}

func Int64Ptr(i int64) *int64 {
	return &i
}
