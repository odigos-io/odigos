package services

import (
	"context"
	"encoding/json"
	"fmt"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	apiactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	graphstatus "github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deriveTypeFromAction(action *model.Action, crd *v1alpha1.Action) model.ActionType {
	// DbQueryTemplatization and InferDbAttributes share ActionFields.scopes, and
	// InferDbAttributes may have empty fields when scoped to all sources — derive
	// these from the CRD config rather than field presence.
	if crd.Spec.DbQueryTemplatization != nil {
		return model.ActionTypeDbQueryTemplatization
	}
	if crd.Spec.InferDbAttributes != nil {
		return model.ActionTypeInferDbAttributes
	}
	if action.Fields.CollectContainerAttributes != nil || action.Fields.CollectReplicaSetAttributes != nil || action.Fields.CollectWorkloadID != nil || action.Fields.CollectClusterID != nil || action.Fields.LabelsAttributes != nil || action.Fields.AnnotationsAttributes != nil {
		return model.ActionTypeK8sAttributesResolver
	}
	if action.Fields.ClusterAttributes != nil || action.Fields.OverwriteExistingValues != nil {
		return model.ActionTypeAddClusterInfo
	}
	if action.Fields.AttributeNamesToDelete != nil {
		return model.ActionTypeDeleteAttribute
	}
	if action.Fields.Renames != nil {
		return model.ActionTypeRenameAttribute
	}
	if action.Fields.PiiCategories != nil || action.Fields.CustomFormatMaskings != nil || action.Fields.CustomRegexMaskings != nil {
		return model.ActionTypePiiMasking
	}
	if crd.Spec.URLTemplatization != nil {
		return model.ActionTypeURLTemplatization
	}
	if action.Fields.URLTemplatizationRulesGroups != nil {
		return model.ActionTypeURLTemplatization
	}
	if action.Fields.ExtractAttribute != nil && len(action.Fields.ExtractAttribute.Extractions) > 0 {
		return model.ActionTypeExtractAttribute
	}

	return model.ActionTypeUnknownType
}

func GetAction(ctx context.Context, id string) (*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	action, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("action with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get action: %v", err)
	}

	convertedAction, err := convertActionToModel(action)
	if err != nil {
		return nil, fmt.Errorf("failed to convert action to model: %v", err)
	}
	return convertedAction, nil
}

func GetActions(ctx context.Context) ([]*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	actions, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %v", err)
	}

	var response []*model.Action
	for _, action := range actions.Items {
		convertedAction, err := convertActionToModel(&action)
		if err != nil {
			return nil, fmt.Errorf("failed to convert action to model: %v", err)
		}
		response = append(response, convertedAction)
	}

	return response, nil
}

func CreateAction(ctx context.Context, input model.ActionInput) (*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	spec, err := getSpecFromInput(input, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec from input: %v", err)
	}

	payload := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "action-",
			Labels: map[string]string{
				k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue,
			},
		},
		Spec: *spec,
	}

	createdAction, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Create(ctx, payload, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %v", err)
	}

	response, err := convertActionToModel(createdAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert action to model: %v", err)
	}

	return response, nil
}

func UpdateAction(ctx context.Context, id string, input model.ActionInput) (*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Get(ctx, id, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to fetch action: %v", err)
	}

	spec, err := getSpecFromInput(input, existingAction)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec from input: %v", err)
	}
	existingAction.Spec = *spec

	updatedAction, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update action: %v", err)
	}

	response, err := convertActionToModel(updatedAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert action to model: %v", err)
	}

	return response, nil
}

func DeleteAction(ctx context.Context, id string) (bool, error) {
	odigosNs := env.GetCurrentNamespace()

	err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("action with ID %s not found", id)
		}
		return false, fmt.Errorf("failed to delete action: %v", err)
	}

	return true, nil
}

func getSpecFromInput(input model.ActionInput, existingAction *v1alpha1.Action) (*v1alpha1.ActionSpec, error) {
	var spec v1alpha1.ActionSpec
	if existingAction != nil {
		spec = existingAction.Spec
	} else {
		spec = v1alpha1.ActionSpec{}
	}

	signals, err := ConvertSignals(input.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	spec.ActionName = DerefString(input.Name)
	spec.Notes = DerefString(input.Notes)
	spec.Disabled = input.Disabled
	spec.Signals = signals

	spec.K8sAttributes = convertK8sAttributesFromInput(input.Fields, existingAction)
	spec.AddClusterInfo = convertAddClusterInfoFromInput(input.Fields, existingAction)
	spec.DeleteAttribute = convertDeleteAttributeFromInput(input.Fields, existingAction)

	renameAttribute, err := convertRenameAttributeFromInput(input.Fields, existingAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rename attribute: %v", err)
	}
	spec.RenameAttribute = renameAttribute

	piiMasking, err := convertPiiMaskingFromInput(input.Fields, existingAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pii masking: %v", err)
	}
	spec.PiiMasking = piiMasking
	spec.URLTemplatization = convertUrlTemplatizationFromInput(input.Fields, existingAction)
	spec.ExtractAttribute = convertExtractAttributeFromInput(input.Fields, existingAction)
	spec.DbQueryTemplatization = convertDbQueryTemplatizationFromInput(input.Type, input.Fields, existingAction)
	spec.InferDbAttributes = convertInferDbAttributesFromInput(input.Type, input.Fields, existingAction)

	return &spec, nil
}

func convertK8sAttributesFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.K8sAttributesConfig {
	withK8sAttributes := details.CollectContainerAttributes != nil ||
		details.CollectReplicaSetAttributes != nil ||
		details.CollectWorkloadID != nil ||
		details.CollectClusterID != nil ||
		details.LabelsAttributes != nil ||
		details.AnnotationsAttributes != nil

	if !withK8sAttributes {
		if existingAction != nil && existingAction.Spec.K8sAttributes != nil {
			return existingAction.Spec.K8sAttributes
		}
		return nil
	}

	// Preserve omitted sibling fields on partial GraphQL updates (same pattern as PiiMasking).
	config := &actionsv1.K8sAttributesConfig{}
	if existingAction != nil && existingAction.Spec.K8sAttributes != nil {
		config = existingAction.Spec.K8sAttributes.DeepCopy()
	}

	if details.CollectContainerAttributes != nil {
		config.CollectContainerAttributes = *details.CollectContainerAttributes
	}
	if details.CollectReplicaSetAttributes != nil {
		config.CollectReplicaSetAttributes = *details.CollectReplicaSetAttributes
	}
	if details.CollectWorkloadID != nil {
		config.CollectWorkloadUID = *details.CollectWorkloadID
	}
	if details.CollectClusterID != nil {
		config.CollectClusterUID = *details.CollectClusterID
	}
	if details.LabelsAttributes != nil {
		config.LabelsAttributes = make([]actionsv1.K8sLabelAttribute, len(details.LabelsAttributes))
		for i, attr := range details.LabelsAttributes {
			config.LabelsAttributes[i] = actionsv1.K8sLabelAttribute{
				LabelKey:     attr.LabelKey,
				AttributeKey: attr.AttributeKey,
			}
			if attr.From != nil {
				from := actionsv1.K8sAttributeSource(*attr.From)
				config.LabelsAttributes[i].From = &from
			}
			if len(attr.FromSources) > 0 {
				config.LabelsAttributes[i].FromSources = make([]actionsv1.K8sAttributeSource, len(attr.FromSources))
				for j, source := range attr.FromSources {
					config.LabelsAttributes[i].FromSources[j] = actionsv1.K8sAttributeSource(source)
				}
			}
		}
	}
	if details.AnnotationsAttributes != nil {
		config.AnnotationsAttributes = make([]actionsv1.K8sAnnotationAttribute, len(details.AnnotationsAttributes))
		for i, attr := range details.AnnotationsAttributes {
			config.AnnotationsAttributes[i] = actionsv1.K8sAnnotationAttribute{
				AnnotationKey: attr.AnnotationKey,
				AttributeKey:  attr.AttributeKey,
			}
			if attr.From != nil {
				from := string(*attr.From)
				config.AnnotationsAttributes[i].From = &from
			}
			if len(attr.FromSources) > 0 {
				config.AnnotationsAttributes[i].FromSources = make([]actionsv1.K8sAttributeSource, len(attr.FromSources))
				for j, source := range attr.FromSources {
					config.AnnotationsAttributes[i].FromSources[j] = actionsv1.K8sAttributeSource(source)
				}
			}
		}
	}

	return config
}

func convertAddClusterInfoFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.AddClusterInfoConfig {
	withAddClusterInfo := details.ClusterAttributes != nil || details.OverwriteExistingValues != nil

	if !withAddClusterInfo {
		if existingAction != nil && existingAction.Spec.AddClusterInfo != nil {
			return existingAction.Spec.AddClusterInfo
		}
		return nil
	}

	// Preserve omitted sibling fields on partial GraphQL updates (same pattern as PiiMasking).
	config := &actionsv1.AddClusterInfoConfig{}
	if existingAction != nil && existingAction.Spec.AddClusterInfo != nil {
		config = existingAction.Spec.AddClusterInfo.DeepCopy()
	}

	if details.ClusterAttributes != nil {
		config.ClusterAttributes = make([]actionsv1.OtelAttributeWithValue, len(details.ClusterAttributes))
		for i, attr := range details.ClusterAttributes {
			config.ClusterAttributes[i] = actionsv1.OtelAttributeWithValue{
				AttributeName:        attr.AttributeName,
				AttributeStringValue: &attr.AttributeStringValue,
			}
		}
	}
	if details.OverwriteExistingValues != nil {
		config.OverwriteExistingValues = *details.OverwriteExistingValues
	}

	return config
}

func convertDeleteAttributeFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.DeleteAttributeConfig {
	withDeleteAttribute := false
	var config *actionsv1.DeleteAttributeConfig

	if details.AttributeNamesToDelete != nil {
		config = &actionsv1.DeleteAttributeConfig{
			AttributeNamesToDelete: details.AttributeNamesToDelete,
		}
		withDeleteAttribute = true
	}

	if !withDeleteAttribute {
		if existingAction != nil && existingAction.Spec.DeleteAttribute != nil {
			return existingAction.Spec.DeleteAttribute
		}
		return nil
	}

	return config
}

func convertRenameAttributeFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) (*actionsv1.RenameAttributeConfig, error) {
	withRenameAttribute := false
	var config *actionsv1.RenameAttributeConfig

	if details.Renames != nil {
		config = &actionsv1.RenameAttributeConfig{}

		var renamesMap map[string]string
		err := json.Unmarshal([]byte(*details.Renames), &renamesMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal renames: %v", err)
		}
		config.Renames = renamesMap
		withRenameAttribute = true
	}

	if !withRenameAttribute {
		if existingAction != nil && existingAction.Spec.RenameAttribute != nil {
			return existingAction.Spec.RenameAttribute, nil
		}
		return nil, nil
	}

	return config, nil
}

var supportedPiiCategories = map[actionsapi.PiiCategory]struct{}{
	actionsapi.CreditCardMasking: {},
	actionsapi.EmailMasking:      {},
	actionsapi.JwtMasking:        {},
	actionsapi.UuidMasking:       {},
}

func convertPiiMaskingFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) (*apiactions.PiiMaskingConfig, error) {
	withPiiMasking := details.PiiCategories != nil ||
		details.CustomFormatMaskings != nil ||
		details.CustomRegexMaskings != nil

	if !withPiiMasking {
		if existingAction != nil && existingAction.Spec.PiiMasking != nil {
			return existingAction.Spec.PiiMasking, nil
		}
		return nil, nil
	}

	config := &apiactions.PiiMaskingConfig{}
	if existingAction != nil && existingAction.Spec.PiiMasking != nil {
		// Preserve fields omitted from the input (partial update).
		config = existingAction.Spec.PiiMasking.DeepCopy()
	}

	if details.PiiCategories != nil {
		piiCategories := make([]actionsapi.PiiCategory, len(details.PiiCategories))
		for i, cat := range details.PiiCategories {
			category := actionsapi.PiiCategory(cat)
			if _, ok := supportedPiiCategories[category]; !ok {
				return nil, fmt.Errorf("unsupported pii category %q: supported values: CREDIT_CARD, EMAIL, JWT, UUID", cat)
			}
			piiCategories[i] = category
		}
		config.PiiCategories = piiCategories
	}

	if details.CustomFormatMaskings != nil {
		formatMaskings := make([]actionsapi.CustomFormatMasking, 0, len(details.CustomFormatMaskings))
		for _, m := range details.CustomFormatMaskings {
			if m == nil {
				continue
			}
			formatMaskings = append(formatMaskings, actionsapi.CustomFormatMasking{
				LookupKey:  m.LookupKey,
				DataFormat: actionsapi.DataFormat(m.DataFormat),
			})
		}
		config.CustomFormatMaskings = formatMaskings
	}

	if details.CustomRegexMaskings != nil {
		regexMaskings := make([]actionsapi.CustomRegexMasking, 0, len(details.CustomRegexMaskings))
		for _, m := range details.CustomRegexMaskings {
			if m == nil {
				continue
			}
			regexMaskings = append(regexMaskings, actionsapi.CustomRegexMasking{
				Regex: m.Regex,
			})
		}
		config.CustomRegexMaskings = regexMaskings
	}

	return config, nil
}

func convertActionToModel(action *v1alpha1.Action) (*model.Action, error) {
	var labelAttrs []*model.K8sLabelAttribute
	if action.Spec.K8sAttributes != nil {
		labelAttrs = convertLabelsAttributesToModel(action.Spec.K8sAttributes.LabelsAttributes)
	}

	var annotAttrs []*model.K8sAnnotationAttribute
	if action.Spec.K8sAttributes != nil {
		annotAttrs = convertAnnotationsAttributesToModel(action.Spec.K8sAttributes.AnnotationsAttributes)
	}

	var clustAttrs []*model.ClusterAttribute
	if action.Spec.AddClusterInfo != nil {
		clustAttrs = convertClusterAttributesToModel(action.Spec.AddClusterInfo.ClusterAttributes)
	}

	var renames *string
	if action.Spec.RenameAttribute != nil {
		stringified, err := stringifyMap(action.Spec.RenameAttribute.Renames)
		if err != nil {
			return nil, fmt.Errorf("failed to stringify renames: %v", err)
		}
		renames = &stringified
	}

	var piiCategories []string
	var customFormatMaskings []*model.CustomFormatMasking
	var customRegexMaskings []*model.CustomRegexMasking
	if action.Spec.PiiMasking != nil {
		piiCategories = convertPiiCategoriesToModel(action.Spec.PiiMasking.PiiCategories)
		customFormatMaskings = convertCustomFormatMaskingsToModel(action.Spec.PiiMasking.CustomFormatMaskings)
		customRegexMaskings = convertCustomRegexMaskingsToModel(action.Spec.PiiMasking.CustomRegexMaskings)
	}

	urlTemplatizationGroups := convertUrlTemplatizationToModel(action.Spec.URLTemplatization)
	urlTemplatizationDefaultGroups := convertUrlTemplatizationDefaultToModel(action.Spec.URLTemplatization)
	extractAttribute := convertExtractAttributeToModel(action.Spec.ExtractAttribute)
	scopes, templatizeLiterals, removePostgresCastOperator := convertDbActionFieldsToModel(action)

	responseFields := &model.ActionFields{
		LabelsAttributes:               labelAttrs,
		AnnotationsAttributes:          annotAttrs,
		ClusterAttributes:              clustAttrs,
		Renames:                        renames,
		PiiCategories:                  piiCategories,
		CustomFormatMaskings:           customFormatMaskings,
		CustomRegexMaskings:            customRegexMaskings,
		URLTemplatizationRulesGroups:   urlTemplatizationGroups,
		URLTemplatizationDefaultGroups: urlTemplatizationDefaultGroups,
		ExtractAttribute:               extractAttribute,
		Scopes:                         scopes,
		TemplatizeLiterals:             templatizeLiterals,
		RemovePostgresCastOperator:     removePostgresCastOperator,
	}

	// Handle K8sAttributes fields
	if action.Spec.K8sAttributes != nil {
		responseFields.CollectContainerAttributes = &action.Spec.K8sAttributes.CollectContainerAttributes
		responseFields.CollectReplicaSetAttributes = &action.Spec.K8sAttributes.CollectReplicaSetAttributes
		responseFields.CollectWorkloadID = &action.Spec.K8sAttributes.CollectWorkloadUID
		responseFields.CollectClusterID = &action.Spec.K8sAttributes.CollectClusterUID
	}

	// Handle AddClusterInfo fields
	if action.Spec.AddClusterInfo != nil {
		responseFields.OverwriteExistingValues = &action.Spec.AddClusterInfo.OverwriteExistingValues
	}

	// Handle DeleteAttribute fields
	if action.Spec.DeleteAttribute != nil {
		responseFields.AttributeNamesToDelete = action.Spec.DeleteAttribute.AttributeNamesToDelete
	}

	signals := []model.SignalType{}
	seen := make(map[model.SignalType]bool)
	for _, s := range action.Spec.Signals {
		signal := model.SignalType(s)
		// Deduplicate: only add if not already seen
		if !seen[signal] {
			seen[signal] = true
			signals = append(signals, signal)
		}
	}

	response := &model.Action{
		ID:          action.Name,
		Name:        &action.Spec.ActionName,
		Notes:       &action.Spec.Notes,
		Disabled:    action.Spec.Disabled,
		Signals:     signals,
		Fields:      responseFields,
		UIGenerated: isActionUiGenerated(action),
	}

	response.Type = deriveTypeFromAction(response, action)
	response.Conditions = ConvertConditions(action.Status.Conditions)
	response.Statuses = graphstatus.ConvertActionConditionsToStatuses(action.Status.Conditions, action.Generation)

	return response, nil
}

func isActionUiGenerated(action *v1alpha1.Action) bool {
	if action == nil || action.Labels == nil {
		return false
	}
	return action.Labels[k8sconsts.OdigosProfilesManagedByLabel] == k8sconsts.OdigosUIManagedByValue
}

func convertLabelsAttributesToModel(labelsAttributes []actionsv1.K8sLabelAttribute) []*model.K8sLabelAttribute {
	var result []*model.K8sLabelAttribute

	for _, attr := range labelsAttributes {
		var from *model.K8sAttributesFrom
		if attr.From != nil {
			tmp := model.K8sAttributesFrom(*attr.From)
			from = &tmp
		}
		var fromSources []model.K8sAttributesFrom
		if len(attr.FromSources) > 0 {
			fromSources = make([]model.K8sAttributesFrom, len(attr.FromSources))
			for i, source := range attr.FromSources {
				fromSources[i] = model.K8sAttributesFrom(source)
			}
		}
		result = append(result, &model.K8sLabelAttribute{
			LabelKey:     attr.LabelKey,
			AttributeKey: attr.AttributeKey,
			From:         from,
			FromSources:  fromSources,
		})
	}

	return result
}

func convertAnnotationsAttributesToModel(annotationsAttributes []actionsv1.K8sAnnotationAttribute) []*model.K8sAnnotationAttribute {
	var result []*model.K8sAnnotationAttribute

	for _, attr := range annotationsAttributes {
		var from *model.K8sAttributesFrom
		if attr.From != nil {
			tmp := model.K8sAttributesFrom(*attr.From)
			from = &tmp
		}
		var fromSources []model.K8sAttributesFrom
		if len(attr.FromSources) > 0 {
			fromSources = make([]model.K8sAttributesFrom, len(attr.FromSources))
			for i, source := range attr.FromSources {
				fromSources[i] = model.K8sAttributesFrom(source)
			}
		}
		result = append(result, &model.K8sAnnotationAttribute{
			AnnotationKey: attr.AnnotationKey,
			AttributeKey:  attr.AttributeKey,
			From:          from,
			FromSources:   fromSources,
		})
	}

	return result
}

func convertClusterAttributesToModel(clusterAttributes []actionsv1.OtelAttributeWithValue) []*model.ClusterAttribute {
	var result []*model.ClusterAttribute

	for _, attr := range clusterAttributes {
		var stringValue string
		if attr.AttributeStringValue != nil {
			stringValue = *attr.AttributeStringValue
		}
		result = append(result, &model.ClusterAttribute{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: stringValue,
		})
	}

	return result
}

func convertPiiCategoriesToModel(piiCategories []actionsapi.PiiCategory) []string {
	var result []string

	for _, category := range piiCategories {
		result = append(result, string(category))
	}

	return result
}

func convertCustomFormatMaskingsToModel(maskings []actionsapi.CustomFormatMasking) []*model.CustomFormatMasking {
	if len(maskings) == 0 {
		return nil
	}
	result := make([]*model.CustomFormatMasking, 0, len(maskings))
	for _, m := range maskings {
		result = append(result, &model.CustomFormatMasking{
			LookupKey:  m.LookupKey,
			DataFormat: model.ExtractionDataFormat(m.DataFormat),
		})
	}
	return result
}

func convertCustomRegexMaskingsToModel(maskings []actionsapi.CustomRegexMasking) []*model.CustomRegexMasking {
	if len(maskings) == 0 {
		return nil
	}
	result := make([]*model.CustomRegexMasking, 0, len(maskings))
	for _, m := range maskings {
		result = append(result, &model.CustomRegexMasking{
			Regex: m.Regex,
		})
	}
	return result
}

func stringifyMap(m map[string]string) (string, error) {
	json, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal map: %v", err)
	}
	return string(json), nil
}

func convertUrlTemplatizationFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *apiactions.URLTemplatizationConfig {
	if details.URLTemplatizationRulesGroups == nil && details.URLTemplatizationDefaultGroups == nil {
		if existingAction != nil && existingAction.Spec.URLTemplatization != nil {
			return existingAction.Spec.URLTemplatization
		}
		return nil
	}

	var rules []apiactions.UrlTemplatizationRule
	if details.URLTemplatizationRulesGroups != nil {
		rules = make([]apiactions.UrlTemplatizationRule, 0, len(details.URLTemplatizationRulesGroups))
		for _, g := range details.URLTemplatizationRulesGroups {
			group := apiactions.UrlTemplatizationRule{
				Scopes: SourcesScopesInputToCRD(g.Scopes),
			}

			for _, rule := range g.TemplatizationRules {
				group.Templates = append(group.Templates, rule.Template)
			}
			rules = append(rules, group)
		}
	} else if existingAction != nil && existingAction.Spec.URLTemplatization != nil {
		rules = existingAction.Spec.URLTemplatization.Rules
	}

	config := &apiactions.URLTemplatizationConfig{
		Rules: rules,
	}
	// Prefer explicit default groups from the GraphQL input when provided. Otherwise preserve
	// any YAML-managed default templatization from the existing Action so that updating
	// rules through the UI does not silently drop it.
	if details.URLTemplatizationDefaultGroups != nil {
		config.Default = convertUrlTemplatizationDefaultFromInput(details.URLTemplatizationDefaultGroups)
	} else if existingAction != nil && existingAction.Spec.URLTemplatization != nil {
		config.Default = existingAction.Spec.URLTemplatization.Default
	}
	return config
}

func convertUrlTemplatizationDefaultFromInput(groups []*model.URLTemplatizationDefaultGroupInput) []apiactions.URLTemplatizationDefaultTemplatizationGroup {
	if groups == nil {
		return nil
	}
	out := make([]apiactions.URLTemplatizationDefaultTemplatizationGroup, 0, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		group := apiactions.URLTemplatizationDefaultTemplatizationGroup{
			Scopes: SourcesScopesInputToCRD(g.Scopes),
		}
		if g.Disabled != nil {
			group.Disabled = *g.Disabled
		}
		if g.SkipPolicy != nil {
			skip := &actionsapi.DefaultTemplatizationSkipPolicyConfig{}
			if g.SkipPolicy.SkipForNonSuccessCodes != nil {
				skip.SkipForNonSuccessCodes = *g.SkipPolicy.SkipForNonSuccessCodes
			}
			if len(g.SkipPolicy.SkipHTTPStatusCodes) > 0 {
				skip.SkipHttpStatusCodes = append([]int(nil), g.SkipPolicy.SkipHTTPStatusCodes...)
			}
			group.SkipPolicy = skip
		}
		out = append(out, group)
	}
	return out
}

func convertUrlTemplatizationToModel(cfg *apiactions.URLTemplatizationConfig) []*model.URLTemplatizationRulesGroup {
	if cfg == nil {
		return nil
	}

	var result []*model.URLTemplatizationRulesGroup
	for _, g := range cfg.Rules {
		group := &model.URLTemplatizationRulesGroup{
			Scopes: SourcesScopesCRDToModel(g.Scopes),
		}

		for _, rule := range g.Templates {
			group.TemplatizationRules = append(group.TemplatizationRules, &model.URLTemplatizationRule{
				Template: rule,
			})
		}
		result = append(result, group)
	}
	return result
}

func convertUrlTemplatizationDefaultToModel(cfg *apiactions.URLTemplatizationConfig) []*model.URLTemplatizationDefaultGroup {
	if cfg == nil || len(cfg.Default) == 0 {
		return nil
	}

	result := make([]*model.URLTemplatizationDefaultGroup, 0, len(cfg.Default))
	for _, g := range cfg.Default {
		group := &model.URLTemplatizationDefaultGroup{
			Scopes: SourcesScopesCRDToModel(g.Scopes),
		}
		if g.Disabled {
			disabled := true
			group.Disabled = &disabled
		}
		if g.SkipPolicy != nil {
			skipPolicy := &model.URLTemplatizationDefaultSkipPolicy{}
			if g.SkipPolicy.SkipForNonSuccessCodes {
				v := true
				skipPolicy.SkipForNonSuccessCodes = &v
			}
			if len(g.SkipPolicy.SkipHttpStatusCodes) > 0 {
				skipPolicy.SkipHTTPStatusCodes = append([]int(nil), g.SkipPolicy.SkipHttpStatusCodes...)
			}
			group.SkipPolicy = skipPolicy
		}
		result = append(result, group)
	}
	return result
}

func convertExtractAttributeFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *apiactions.ExtractAttributeConfig {
	if details.ExtractAttribute == nil {
		if existingAction != nil && existingAction.Spec.ExtractAttribute != nil {
			return existingAction.Spec.ExtractAttribute
		}
		return nil
	}

	extractions := make([]actionsapi.Extraction, 0, len(details.ExtractAttribute.Extractions))
	for _, e := range details.ExtractAttribute.Extractions {
		row := actionsapi.Extraction{
			TargetAttributeName: e.TargetAttributeName,
			LookupKey:           DerefString(e.LookupKey),
			Regex:               DerefString(e.Regex),
		}
		if e.DataFormat != nil {
			row.DataFormat = actionsapi.DataFormat(*e.DataFormat)
		}
		extractions = append(extractions, row)
	}

	return &apiactions.ExtractAttributeConfig{
		ExtractAttributeConfig: actionsapi.ExtractAttributeConfig{
			Extractions: extractions,
		},
	}
}

func convertExtractAttributeToModel(cfg *apiactions.ExtractAttributeConfig) *model.ExtractAttribute {
	if cfg == nil {
		return nil
	}

	extractions := make([]*model.Extraction, 0, len(cfg.Extractions))
	for _, e := range cfg.Extractions {
		row := &model.Extraction{
			TargetAttributeName: e.TargetAttributeName,
		}
		if e.LookupKey != "" {
			lookupKey := e.LookupKey
			row.LookupKey = &lookupKey
		}
		if e.DataFormat != "" {
			df := model.ExtractionDataFormat(e.DataFormat)
			row.DataFormat = &df
		}
		if e.Regex != "" {
			regex := e.Regex
			row.Regex = &regex
		}
		extractions = append(extractions, row)
	}

	return &model.ExtractAttribute{
		Extractions: extractions,
	}
}

func convertDbQueryTemplatizationFromInput(actionType model.ActionType, details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *apiactions.DbQueryTemplatizationConfig {
	if actionType != model.ActionTypeDbQueryTemplatization {
		return nil
	}

	config := &apiactions.DbQueryTemplatizationConfig{}
	if details.Scopes != nil {
		config.Scopes = SourcesScopesInputToCRD(details.Scopes)
	} else if existingAction != nil && existingAction.Spec.DbQueryTemplatization != nil {
		// Preserve scopes when GraphQL omits them (partial update). Empty scopes
		// input still clears explicitly via SourcesScopesInputToCRD.
		config.Scopes = existingAction.Spec.DbQueryTemplatization.Scopes
	}
	if details.TemplatizeLiterals != nil {
		config.TemplatizeLiterals = *details.TemplatizeLiterals
	} else if existingAction != nil && existingAction.Spec.DbQueryTemplatization != nil {
		config.TemplatizeLiterals = existingAction.Spec.DbQueryTemplatization.TemplatizeLiterals
	}
	if details.RemovePostgresCastOperator != nil {
		config.RemovePostgresCastOperator = *details.RemovePostgresCastOperator
	} else if existingAction != nil && existingAction.Spec.DbQueryTemplatization != nil {
		config.RemovePostgresCastOperator = existingAction.Spec.DbQueryTemplatization.RemovePostgresCastOperator
	}
	return config
}

func convertInferDbAttributesFromInput(actionType model.ActionType, details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *apiactions.InferDbAttributesConfig {
	if actionType != model.ActionTypeInferDbAttributes {
		return nil
	}

	config := &apiactions.InferDbAttributesConfig{}
	if details.Scopes != nil {
		config.Scopes = SourcesScopesInputToCRD(details.Scopes)
	} else if existingAction != nil && existingAction.Spec.InferDbAttributes != nil {
		// Preserve scopes when GraphQL omits them (partial update). Empty scopes
		// input still clears explicitly via SourcesScopesInputToCRD.
		config.Scopes = existingAction.Spec.InferDbAttributes.Scopes
	}
	return config
}

func convertDbActionFieldsToModel(action *v1alpha1.Action) (*model.SourcesScopes, *bool, *bool) {
	if action.Spec.DbQueryTemplatization != nil {
		templatizeLiterals := action.Spec.DbQueryTemplatization.TemplatizeLiterals
		removePostgresCastOperator := action.Spec.DbQueryTemplatization.RemovePostgresCastOperator
		return SourcesScopesCRDToModel(action.Spec.DbQueryTemplatization.Scopes), &templatizeLiterals, &removePostgresCastOperator
	}
	if action.Spec.InferDbAttributes != nil {
		return SourcesScopesCRDToModel(action.Spec.InferDbAttributes.Scopes), nil, nil
	}
	return nil, nil, nil
}
