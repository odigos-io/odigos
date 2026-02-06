package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services/destination_recognition"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	k8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetDestinationCategories() model.GetDestinationCategories {
	var resp model.GetDestinationCategories
	itemsByCategory := make(map[string][]*model.DestinationTypesCategoryItem)

	for _, destConfig := range destinations.Get() {
		item := DestinationTypeConfigToCategoryItem(destConfig)
		itemsByCategory[destConfig.Metadata.Category] = append(itemsByCategory[destConfig.Metadata.Category], &item)
	}

	descriptions := map[string]string{
		"managed":     "Effortless Monitoring with Scalable Performance Management",
		"self hosted": "Full Control and Customization for Advanced Application Monitoring",
	}

	for category, items := range itemsByCategory {
		resp.Categories = append(resp.Categories, &model.DestinationsCategory{
			Name:        category,
			Description: descriptions[category],
			Items:       items,
		})
	}

	return resp
}

func DestinationTypeConfigToCategoryItem(destConfig destinations.Destination) model.DestinationTypesCategoryItem {
	fields := []*model.DestinationFieldYamlProperties{}

	for _, field := range destConfig.Spec.Fields {
		componentPropsJSON, err := json.Marshal(field.ComponentProps)

		var customReadDataLabels []*model.CustomReadDataLabel
		for _, label := range field.CustomReadDataLabels {
			customReadDataLabels = append(customReadDataLabels, &model.CustomReadDataLabel{
				Condition: label.Condition,
				Title:     label.Title,
				Value:     label.Value,
			})
		}

		if err == nil {
			fields = append(fields, &model.DestinationFieldYamlProperties{
				Name:                 field.Name,
				DisplayName:          field.DisplayName,
				ComponentType:        field.ComponentType,
				ComponentProperties:  string(componentPropsJSON),
				Secret:               field.Secret,
				InitialValue:         field.InitialValue,
				RenderCondition:      field.RenderCondition,
				HideFromReadData:     field.HideFromReadData,
				CustomReadDataLabels: customReadDataLabels,
			})
		}
	}

	return model.DestinationTypesCategoryItem{
		Type:                    string(destConfig.Metadata.Type),
		DisplayName:             destConfig.Metadata.DisplayName,
		ImageURL:                GetImageURL(destConfig.Spec.Image),
		TestConnectionSupported: destConfig.Spec.TestConnectionSupported,
		SupportedSignals: &model.SupportedSignals{
			Traces: &model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Traces.Supported,
			},
			Metrics: &model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Metrics.Supported,
			},
			Logs: &model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Logs.Supported,
			},
		},
		Fields: fields,
	}

}

func GetDestinationTypeConfig(destType common.DestinationType) (*destinations.Destination, error) {
	for _, dest := range destinations.Get() {
		if dest.Metadata.Type == destType {
			return &dest, nil
		}
	}

	return nil, fmt.Errorf("destination type %s not found", destType)
}

func VerifyDestinationDataScheme(destType common.DestinationType, destTypeConfig *destinations.Destination, data map[string]string) []error {

	errors := []error{}

	// verify all fields in config are present in data (assuming here all fields are required)
	for _, field := range destTypeConfig.Spec.Fields {
		required, ok := field.ComponentProps["required"].(bool)
		if !ok || !required {
			continue
		}
		fieldValue, found := data[field.Name]
		if !found || fieldValue == "" {
			errors = append(errors, fmt.Errorf("field %s is required", field.Name))
		}
	}

	// verify data fields are found in config
	for dataField := range data {
		found := false
		// iterating all fields in config every time, assuming it's a small list
		for _, field := range destTypeConfig.Spec.Fields {
			if dataField == field.Name {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf("field %s is not found in config for destination type '%s'", dataField, destType))
		}
	}

	return errors
}

func TransformFieldsToDataAndSecrets(destTypeConfig *destinations.Destination, fields map[string]string) (map[string]string, map[string]string) {

	dataFields := map[string]string{}
	secretFields := map[string]string{}

	for fieldName, fieldValue := range fields {

		// it is possible that some fields are not required and are empty.
		// we should treat them as empty
		if fieldValue == "" {
			continue
		}

		// for each field in the data, find it's config
		// assuming the list is small so it's ok to iterate it
		for _, fieldConfig := range destTypeConfig.Spec.Fields {
			if fieldName == fieldConfig.Name {
				if fieldConfig.Secret {
					secretFields[fieldName] = fieldValue
				} else {
					dataFields[fieldName] = fieldValue
				}
			}
		}
	}

	return dataFields, secretFields
}

func GetDestinationSecretFields(c context.Context, odigosns string, dest *v1alpha1.Destination) (map[string]string, error) {

	secretFields := map[string]string{}
	secretRef := dest.Spec.SecretRef

	if secretRef == nil {
		return secretFields, nil
	}

	secret, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Get(c, secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	for k, v := range secret.Data {
		secretFields[k] = string(v)
	}

	return secretFields, nil
}

// ExtractSecretFields extracts string fields from a Secret object.
// Used by batch-fetch paths that already have the secret in memory.
func ExtractSecretFields(secret *k8s.Secret) map[string]string {
	fields := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		fields[k] = string(v)
	}
	return fields
}

func K8sDestinationToEndpointFormat(k8sDest v1alpha1.Destination, secretFields map[string]string) model.Destination {
	destType := k8sDest.Spec.Type
	destName := k8sDest.Spec.DestinationName
	mergedFields := mergeDataAndSecrets(k8sDest.Spec.Data, secretFields)
	dest, _ := destinations.GetDestinationByType(string(destType))
	destTypeConfig := DestinationTypeConfigToCategoryItem(dest)

	fieldsJSON, err := json.Marshal(mergedFields)
	if err != nil {
		// Handle JSON encoding error
		fmt.Printf("Error marshaling fields to JSON: %v\n", err)
		fieldsJSON = []byte("{}") // Set to an empty JSON object in case of error
	}

	var conditions []*model.Condition
	for _, condition := range k8sDest.Status.Conditions {
		var status model.ConditionStatus

		switch condition.Status {
		case metav1.ConditionUnknown:
			status = model.ConditionStatusLoading
		case metav1.ConditionTrue:
			status = model.ConditionStatusSuccess
		case metav1.ConditionFalse:
			status = model.ConditionStatusError
		}

		// force "disabled" status ovverrides for certain "reasons"
		if v1alpha1.IsReasonStatusDisabled(condition.Reason) {
			status = model.ConditionStatusDisabled
		}

		conditions = append(conditions, &model.Condition{
			Status:             status,
			Type:               condition.Type,
			Reason:             &condition.Reason,
			Message:            &condition.Message,
			LastTransitionTime: func(s string) *string { return &s }(condition.LastTransitionTime.String()),
		})
	}

	disabled := false
	if k8sDest.Spec.Disabled != nil {
		disabled = *k8sDest.Spec.Disabled
	}

	return model.Destination{
		ID:              k8sDest.Name,
		Type:            string(destType),
		Name:            destName,
		Disabled:        disabled,
		DataStreamNames: ExtractDataStreamsFromDestination(k8sDest),
		ExportedSignals: &model.ExportedSignals{
			Traces:  isSignalExported(k8sDest, common.TracesObservabilitySignal),
			Metrics: isSignalExported(k8sDest, common.MetricsObservabilitySignal),
			Logs:    isSignalExported(k8sDest, common.LogsObservabilitySignal),
		},
		Fields:          string(fieldsJSON),
		DestinationType: &destTypeConfig,
		Conditions:      conditions,
	}
}

func isSignalExported(dest v1alpha1.Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dest.Spec.Signals {
		if s == signal {
			return true
		}
	}

	return false
}

func mergeDataAndSecrets(data map[string]string, secrets map[string]string) map[string]string {
	merged := map[string]string{}

	for k, v := range data {
		merged[k] = v
	}

	for k, v := range secrets {
		merged[k] = v
	}

	return merged
}
func ExportedSignalsObjectToSlice(signals *model.ExportedSignalsInput) []common.ObservabilitySignal {
	var resp []common.ObservabilitySignal
	if signals.Traces {
		resp = append(resp, common.TracesObservabilitySignal)
	}
	if signals.Metrics {
		resp = append(resp, common.MetricsObservabilitySignal)
	}
	if signals.Logs {
		resp = append(resp, common.LogsObservabilitySignal)
	}

	return resp
}

func CreateDestinationSecret(ctx context.Context, destType common.DestinationType, secretFields map[string]string, ns string) (*k8s.LocalObjectReference, error) {
	generateNamePrefix := "odigos.io.dest." + string(destType) + "-"
	secret := k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
		},
		StringData: secretFields,
	}
	newSecret, err := CreateResourceWithGenerateName(ctx, func() (*k8s.Secret, error) {
		return kube.DefaultClient.CoreV1().Secrets(ns).Create(ctx, &secret, metav1.CreateOptions{})
	})
	if err != nil {
		return nil, err
	}
	return &k8s.LocalObjectReference{
		Name: newSecret.Name,
	}, nil
}

func AddDestinationOwnerReferenceToSecret(ctx context.Context, ns string, dest *v1alpha1.Destination) error {
	destOwnerRef := metav1.OwnerReference{
		APIVersion: "odigos.io/v1alpha1",
		Kind:       "Destination",
		Name:       dest.Name,
		UID:        dest.UID,
	}

	secretPatch := []struct {
		Op    string                  `json:"op"`
		Path  string                  `json:"path"`
		Value []metav1.OwnerReference `json:"value"`
	}{{
		Op:    "add",
		Path:  "/metadata/ownerReferences",
		Value: []metav1.OwnerReference{destOwnerRef},
	},
	}

	secretPatchBytes, err := json.Marshal(secretPatch)
	if err != nil {
		return err
	}

	_, err = kube.DefaultClient.CoreV1().Secrets(ns).Patch(ctx, dest.Spec.SecretRef.Name, types.JSONPatchType, secretPatchBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func PotentialDestinations(ctx context.Context) []destination_recognition.DestinationDetails {
	ns := env.GetCurrentNamespace()

	relevantNamespaces, err := getRelevantNameSpaces(ctx, ns)
	if err != nil {
		return nil
	}

	// Existing Destinations
	existingDestination, err := kube.DefaultClient.OdigosClient.Destinations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	destinationDetails, err := destination_recognition.GetAllPotentialDestinationDetails(ctx, relevantNamespaces, existingDestination)
	if err != nil {
		return nil
	}

	return destinationDetails
}

func deleteDestinationAndSecret(ctx context.Context, destination *v1alpha1.Destination) error {
	ns := env.GetCurrentNamespace()

	// Delete the destination
	err := kube.DefaultClient.OdigosClient.Destinations(ns).Delete(ctx, destination.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete destination: %w", err)
	}

	// If the destination has a secret, delete it as well
	if destination.Spec.SecretRef != nil {
		err = kube.DefaultClient.CoreV1().Secrets(ns).Delete(ctx, destination.Spec.SecretRef.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
	}

	return nil
}

func UpdateDestination(ctx context.Context, destination *v1alpha1.Destination) error {
	ns := env.GetCurrentNamespace()

	// Update the destination
	_, err := kube.DefaultClient.OdigosClient.Destinations(ns).Update(ctx, destination, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update destination: %w", err)
	}

	return nil
}
