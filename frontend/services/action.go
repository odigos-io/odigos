package services

import (
	"context"
	"encoding/json"
	"fmt"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urlactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deriveTypeFromAction(action *model.Action) model.ActionType {
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
	if action.Fields.PiiCategories != nil {
		return model.ActionTypePiiMasking
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

	spec.PiiMasking = convertPiiMaskingFromInput(input.Fields, existingAction)
	spec.URLTemplatization = convertUrlTemplatizationFromInput(input.Fields, existingAction)
	spec.ExtractAttribute = convertExtractAttributeFromInput(input.Fields, existingAction)

	return &spec, nil
}

func convertK8sAttributesFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.K8sAttributesConfig {
	withK8sAttributes := false
	var config *actionsv1.K8sAttributesConfig

	if details.CollectContainerAttributes != nil ||
		details.CollectReplicaSetAttributes != nil ||
		details.CollectWorkloadID != nil ||
		details.CollectClusterID != nil ||
		details.LabelsAttributes != nil ||
		details.AnnotationsAttributes != nil {

		config = &actionsv1.K8sAttributesConfig{}

		if details.CollectContainerAttributes != nil {
			config.CollectContainerAttributes = *details.CollectContainerAttributes
			withK8sAttributes = true
		}
		if details.CollectReplicaSetAttributes != nil {
			config.CollectReplicaSetAttributes = *details.CollectReplicaSetAttributes
			withK8sAttributes = true
		}
		if details.CollectWorkloadID != nil {
			config.CollectWorkloadUID = *details.CollectWorkloadID
			withK8sAttributes = true
		}
		if details.CollectClusterID != nil {
			config.CollectClusterUID = *details.CollectClusterID
			withK8sAttributes = true
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
			withK8sAttributes = true
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
			withK8sAttributes = true
		}
	}

	if !withK8sAttributes {
		if existingAction != nil && existingAction.Spec.K8sAttributes != nil {
			return existingAction.Spec.K8sAttributes
		}
		return nil
	}

	return config
}

func convertAddClusterInfoFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.AddClusterInfoConfig {
	withAddClusterInfo := false
	var config *actionsv1.AddClusterInfoConfig

	if details.ClusterAttributes != nil || details.OverwriteExistingValues != nil {
		config = &actionsv1.AddClusterInfoConfig{}

		if details.ClusterAttributes != nil {
			config.ClusterAttributes = make([]actionsv1.OtelAttributeWithValue, len(details.ClusterAttributes))
			for i, attr := range details.ClusterAttributes {
				config.ClusterAttributes[i] = actionsv1.OtelAttributeWithValue{
					AttributeName:        attr.AttributeName,
					AttributeStringValue: &attr.AttributeStringValue,
				}
			}
			withAddClusterInfo = true
		}
		if details.OverwriteExistingValues != nil {
			config.OverwriteExistingValues = *details.OverwriteExistingValues
			withAddClusterInfo = true
		}
	}

	if !withAddClusterInfo {
		if existingAction != nil && existingAction.Spec.AddClusterInfo != nil {
			return existingAction.Spec.AddClusterInfo
		}
		return nil
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

func convertPiiMaskingFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.PiiMaskingConfig {
	withPiiMasking := false
	var config *actionsv1.PiiMaskingConfig

	if details.PiiCategories != nil {
		config = &actionsv1.PiiMaskingConfig{}

		piiCategories := make([]actionsv1.PiiCategory, len(details.PiiCategories))
		for i, cat := range details.PiiCategories {
			piiCategories[i] = actionsv1.PiiCategory(cat)
		}
		config.PiiCategories = piiCategories
		withPiiMasking = true
	}

	if !withPiiMasking {
		if existingAction != nil && existingAction.Spec.PiiMasking != nil {
			return existingAction.Spec.PiiMasking
		}
		return nil
	}

	return config
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
	if action.Spec.PiiMasking != nil {
		piiCategories = convertPiiCategoriesToModel(action.Spec.PiiMasking.PiiCategories)
	}

	urlTemplatizationGroups := convertUrlTemplatizationToModel(action.Spec.URLTemplatization)
	extractAttribute := convertExtractAttributeToModel(action.Spec.ExtractAttribute)

	responseFields := &model.ActionFields{
		LabelsAttributes:             labelAttrs,
		AnnotationsAttributes:        annotAttrs,
		ClusterAttributes:            clustAttrs,
		Renames:                      renames,
		PiiCategories:                piiCategories,
		URLTemplatizationRulesGroups: urlTemplatizationGroups,
		ExtractAttribute:             extractAttribute,
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
		ID:       action.Name,
		Name:     &action.Spec.ActionName,
		Notes:    &action.Spec.Notes,
		Disabled: action.Spec.Disabled,
		Signals:  signals,
		Fields:   responseFields,
	}

	response.Type = deriveTypeFromAction(response)
	response.Conditions = ConvertConditions(action.Status.Conditions)

	return response, nil
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

func convertPiiCategoriesToModel(piiCategories []actionsv1.PiiCategory) []string {
	var result []string

	for _, category := range piiCategories {
		result = append(result, string(category))
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

func convertUrlTemplatizationFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *urlactions.URLTemplatizationConfig {
	if details.URLTemplatizationRulesGroups == nil {
		if existingAction != nil && existingAction.Spec.URLTemplatization != nil {
			return existingAction.Spec.URLTemplatization
		}
		return nil
	}

	rules := make([]urlactions.UrlTemplatizationRule, 0, len(details.URLTemplatizationRulesGroups))
	for _, g := range details.URLTemplatizationRulesGroups {
		group := urlactions.UrlTemplatizationRule{}

		// Fold the URL-templatization filter form into the tri-list SourcesScopes shape:
		//   * Each WorkloadFilter row → one PodWorkload appended to Sources (with namespace baked in).
		//   * Singleton fallback path: build one PodWorkload if a workload identity is given,
		//     otherwise fall back to a namespace-only entry.
		//   * FilterProgrammingLanguage → Languages.
		scopes := &k8sconsts.SourcesScopes{}
		if len(g.WorkloadFilters) > 0 {
			for _, wf := range g.WorkloadFilters {
				pw := k8sconsts.PodWorkload{}
				if g.FilterK8sNamespace != nil {
					pw.Namespace = *g.FilterK8sNamespace
				}
				if wf.Kind != nil {
					pw.Kind = k8sconsts.WorkloadKind(*wf.Kind)
				}
				if wf.Name != nil {
					pw.Name = *wf.Name
				}
				scopes.Sources = append(scopes.Sources, pw)
			}
		} else if g.FilterK8sNamespace != nil || g.FilterK8sWorkloadKind != nil || g.FilterK8sWorkloadName != nil {
			if g.FilterK8sWorkloadKind != nil || g.FilterK8sWorkloadName != nil {
				pw := k8sconsts.PodWorkload{}
				if g.FilterK8sNamespace != nil {
					pw.Namespace = *g.FilterK8sNamespace
				}
				if g.FilterK8sWorkloadKind != nil {
					pw.Kind = k8sconsts.WorkloadKind(*g.FilterK8sWorkloadKind)
				}
				if g.FilterK8sWorkloadName != nil {
					pw.Name = *g.FilterK8sWorkloadName
				}
				scopes.Sources = append(scopes.Sources, pw)
			} else if g.FilterK8sNamespace != nil {
				scopes.Namespaces = append(scopes.Namespaces, *g.FilterK8sNamespace)
			}
		}
		if g.FilterProgrammingLanguage != nil {
			scopes.Languages = append(scopes.Languages, common.ProgrammingLanguage(*g.FilterProgrammingLanguage))
		}
		if len(scopes.Sources) > 0 || len(scopes.Namespaces) > 0 || len(scopes.Languages) > 0 {
			group.Scopes = scopes
		}

		for _, rule := range g.TemplatizationRules {
			group.Templates = append(group.Templates, rule.Template)
		}
		rules = append(rules, group)
	}

	urlTemplatization := &urlactions.URLTemplatizationConfig{
		Rules: rules,
	}
	if existingAction != nil && existingAction.Spec.URLTemplatization != nil {
		urlTemplatization.Default = existingAction.Spec.URLTemplatization.Default
	}
	return urlTemplatization
}

func convertUrlTemplatizationToModel(cfg *urlactions.URLTemplatizationConfig) []*model.URLTemplatizationRulesGroup {
	if cfg == nil {
		return nil
	}

	var result []*model.URLTemplatizationRulesGroup
	for _, g := range cfg.Rules {
		group := &model.URLTemplatizationRulesGroup{}

		// Unfold tri-list SourcesScopes back into the URL-templatization filter form.
		// The GraphQL shape exposes only single-value FilterK8sNamespace and
		// FilterProgrammingLanguage, so multi-namespace/multi-language scopes are
		// projected to the first entry (best-effort; the wire format predates the list).
		if g.Scopes != nil {
			for _, src := range g.Scopes.Sources {
				if src.Kind != "" || src.Name != "" {
					filter := &model.TemplatizationWorkloadFilter{}
					if src.Kind != "" {
						kind := model.K8sResourceKind(src.Kind)
						filter.Kind = &kind
					}
					if src.Name != "" {
						name := src.Name
						filter.Name = &name
					}
					group.WorkloadFilters = append(group.WorkloadFilters, filter)
				}
				if src.Namespace != "" && group.FilterK8sNamespace == nil {
					ns := src.Namespace
					group.FilterK8sNamespace = &ns
				}
			}
			if group.FilterK8sNamespace == nil && len(g.Scopes.Namespaces) > 0 {
				ns := g.Scopes.Namespaces[0]
				group.FilterK8sNamespace = &ns
			}
			if len(g.Scopes.Languages) > 0 {
				lang := string(g.Scopes.Languages[0])
				group.FilterProgrammingLanguage = &lang
			}
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

func convertExtractAttributeFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *urlactions.ExtractAttributeConfig {
	if details.ExtractAttribute == nil {
		if existingAction != nil && existingAction.Spec.ExtractAttribute != nil {
			return existingAction.Spec.ExtractAttribute
		}
		return nil
	}

	extractions := make([]urlactions.Extraction, 0, len(details.ExtractAttribute.Extractions))
	for _, e := range details.ExtractAttribute.Extractions {
		row := urlactions.Extraction{
			TargetAttributeName: e.TargetAttributeName,
			LookupKey:           DerefString(e.LookupKey),
			Regex:               DerefString(e.Regex),
		}
		if e.DataFormat != nil {
			row.DataFormat = urlactions.DataFormat(*e.DataFormat)
		}
		extractions = append(extractions, row)
	}

	return &urlactions.ExtractAttributeConfig{
		Extractions: extractions,
	}
}

func convertExtractAttributeToModel(cfg *urlactions.ExtractAttributeConfig) *model.ExtractAttribute {
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
