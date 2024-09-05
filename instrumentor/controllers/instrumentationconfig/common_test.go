package instrumentationconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateInstrumentationConfigForWorkload_SingleLanguage(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{{
				ContainerName: "test-container",
				Language:      common.JavascriptProgrammingLanguage,
			}},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 2 {
		t.Errorf("Expected 2 sdk configs, got %d", len(ic.Spec.SdkConfigs))
	}
	if ic.Spec.SdkConfigs[0].Language != common.JavascriptProgrammingLanguage {
		t.Errorf("Expected language %s, got %s", common.JavascriptProgrammingLanguage, ic.Spec.SdkConfigs[0].Language)
	}
	if ic.Spec.SdkConfigs[1].Language != common.PythonProgrammingLanguage {
		t.Errorf("Expected language %s, got %s", common.PythonProgrammingLanguage, ic.Spec.SdkConfigs[1].Language)
	}
}

func TestUpdateInstrumentationConfigForWorkload_IgnoreUnknownLanguage(t *testing.T) {

	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test-container-1",
					Language:      common.JavascriptProgrammingLanguage,
				},
				{
					ContainerName: "test-container-2",
					Language:      common.UnknownProgrammingLanguage,
				},
				{
					ContainerName: "test-container-3",
					Language:      common.IgnoredProgrammingLanguage,
				},
			},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{},
		},
	}
	rules := &odigosv1.InstrumentationRuleList{}
	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes:           &[]string{"application/json"},
							MaxPayloadLength:    Int64Ptr(1234),
							DropPartialPayloads: BoolPtr(true),
						},
					},
				},
			},
		},
	}
	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					Workloads: &[]workload.PodWorkload{
						{
							Name:      "test",
							Kind:      workload.WorkloadKindDeployment,
							Namespace: "testns",
						},
					},
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					Workloads: &[]workload.PodWorkload{
						{
							Name:      "someotherdeployment",
							Kind:      workload.WorkloadKindDeployment,
							Namespace: "testns",
						},
					},
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes:           &[]string{"application/json", "application/text"},
							MaxPayloadLength:    Int64Ptr(1111),
							DropPartialPayloads: BoolPtr(true),
						},
					},
				},
			},
			{
				Spec: odigosv1.InstrumentationRuleSpec{
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes:           &[]string{"application/xml", "application/json"},
							MaxPayloadLength:    Int64Ptr(2222),
							DropPartialPayloads: BoolPtr(false),
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(ic.Spec.SdkConfigs) != 1 {
		t.Errorf("Expected 1 sdk config, got %d", len(ic.Spec.SdkConfigs))
	}

	// mime types should merge
	if len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes) != 3 {
		t.Errorf("Expected 2 mime types, got %d", len(*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes))
	}
	if (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[0] != "application/json" {
		t.Errorf("Expected mime type %s, got %s", "application/json", (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[0])
	}
	if (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[1] != "application/text" {
		t.Errorf("Expected mime type %s, got %s", "application/text", (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[1])
	}
	if (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[2] != "application/xml" {
		t.Errorf("Expected mime type %s, got %s", "application/xml", (*ic.Spec.SdkConfigs[0].DefaultPayloadCollection.HttpRequest.MimeTypes)[1])
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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
		Spec: odigosv1.InstrumentationConfigSpec{},
	}
	ia := odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "testns",
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
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
					PayloadCollection: &odigosv1.PayloadCollection{
						HttpRequest: &odigosv1.HttpPayloadCollectionRule{
							MimeTypes: &[]string{"application/json"},
						},
					},
				},
			},
		},
	}

	err := updateInstrumentationConfigForWorkload(&ic, &ia, rules)
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

func TestMergeHttpPayloadCollectionRules(t *testing.T) {
	res := mergeHttpPayloadCollectionRules(&odigosv1.HttpPayloadCollectionRule{
		MimeTypes:        &[]string{"application/json"},
		MaxPayloadLength: Int64Ptr(1234),
	}, &odigosv1.HttpPayloadCollectionRule{
		MimeTypes:           &[]string{"application/xml"},
		DropPartialPayloads: BoolPtr(false),
	})
	if len(*res.MimeTypes) != 2 {
		t.Errorf("Expected 2 mime types, got %d", len(*res.MimeTypes))
	}
	if (*res.MimeTypes)[0] != "application/json" {
		t.Errorf("Expected mime type %s, got %s", "application/json", (*res.MimeTypes)[0])
	}
	if (*res.MimeTypes)[1] != "application/xml" {
		t.Errorf("Expected mime type %s, got %s", "application/xml", (*res.MimeTypes)[1])
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
	res := mergeHttpPayloadCollectionRules(nil, &odigosv1.HttpPayloadCollectionRule{
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
	res := mergeHttpPayloadCollectionRules(&odigosv1.HttpPayloadCollectionRule{
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
