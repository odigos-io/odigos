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

func GetDestinationTypes() model.GetDestinationTypesResponse {
	var resp model.GetDestinationTypesResponse
	itemsByCategory := make(map[string][]model.DestinationTypesCategoryItem)
	for _, destConfig := range destinations.Get() {
		item := DestinationTypeConfigToCategoryItem(destConfig)
		itemsByCategory[destConfig.Metadata.Category] = append(itemsByCategory[destConfig.Metadata.Category], item)
	}

	for category, items := range itemsByCategory {
		resp.Categories = append(resp.Categories, model.DestinationsCategory{
			Name:  category,
			Items: items,
		})

	}

	return resp

}

func DestinationTypeConfigToCategoryItem(destConfig destinations.Destination) model.DestinationTypesCategoryItem {

	return model.DestinationTypesCategoryItem{
		Type:                    string(destConfig.Metadata.Type),
		DisplayName:             destConfig.Metadata.DisplayName,
		ImageUrl:                GetImageURL(destConfig.Spec.Image),
		TestConnectionSupported: destConfig.Spec.TestConnectionSupported,
		SupportedSignals: model.SupportedSignals{
			Traces: model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Traces.Supported,
			},
			Metrics: model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Metrics.Supported,
			},
			Logs: model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Logs.Supported,
			},
		},
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

func K8sDestinationToEndpointFormat(k8sDest v1alpha1.Destination, secretFields map[string]string) model.Destination {
	destType := k8sDest.Spec.Type
	destName := k8sDest.Spec.DestinationName
	mergedFields := mergeDataAndSecrets(k8sDest.Spec.Data, secretFields)
	destTypeConfig := DestinationTypeConfigToCategoryItem(destinations.GetDestinationByType(string(destType)))

	fieldsJSON, err := json.Marshal(mergedFields)
	if err != nil {
		// Handle JSON encoding error
		fmt.Printf("Error marshaling fields to JSON: %v\n", err)
		fieldsJSON = []byte("{}") // Set to an empty JSON object in case of error
	}

	var conditions []metav1.Condition
	for _, condition := range k8sDest.Status.Conditions {
		conditions = append(conditions, metav1.Condition{
			Type:               condition.Type,
			Status:             condition.Status,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime,
		})
	}

	return model.Destination{
		Id:   k8sDest.Name,
		Name: destName,
		Type: destType,
		ExportedSignals: model.ExportedSignals{
			Traces:  isSignalExported(k8sDest, common.TracesObservabilitySignal),
			Metrics: isSignalExported(k8sDest, common.MetricsObservabilitySignal),
			Logs:    isSignalExported(k8sDest, common.LogsObservabilitySignal),
		},
		Fields:          string(fieldsJSON),
		DestinationType: destTypeConfig,
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
	newSecret, err := kube.DefaultClient.CoreV1().Secrets(ns).Create(ctx, &secret, metav1.CreateOptions{})
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

	relevantNamespaces, err := getRelevantNameSpaces(ctx, env.GetCurrentNamespace())
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
