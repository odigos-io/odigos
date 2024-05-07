package endpoints

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/kube"
	k8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetDestinationTypesResponse struct {
	Categories []DestinationsCategory `json:"categories"`
}

type DestinationsCategory struct {
	Name  string                         `json:"name"`
	Items []DestinationTypesCategoryItem `json:"items"`
}

type DestinationTypesCategoryItem struct {
	Type             common.DestinationType `json:"type"`
	DisplayName      string                 `json:"display_name"`
	ImageUrl         string                 `json:"image_url"`
	SupportedSignals SupportedSignals       `json:"supported_signals"`
}

type SupportedSignals struct {
	Traces  ObservabilitySignalSupport `json:"traces"`
	Metrics ObservabilitySignalSupport `json:"metrics"`
	Logs    ObservabilitySignalSupport `json:"logs"`
}

type ObservabilitySignalSupport struct {
	Supported bool `json:"supported"`
}

type ExportedSignals struct {
	Traces  bool `json:"traces"`
	Metrics bool `json:"metrics"`
	Logs    bool `json:"logs"`
}

type Destination struct {
	Id              string                       `json:"id"`
	Name            string                       `json:"name"`
	Type            common.DestinationType       `json:"type"`
	ExportedSignals ExportedSignals              `json:"signals"`
	Fields          map[string]string            `json:"fields"`
	DestinationType DestinationTypesCategoryItem `json:"destination_type"`
}

func GetDestinationTypes(c *gin.Context) {
	var resp GetDestinationTypesResponse
	itemsByCategory := make(map[string][]DestinationTypesCategoryItem)
	for _, destConfig := range destinations.Get() {
		item := DestinationTypeConfigToCategoryItem(destConfig)
		itemsByCategory[destConfig.Metadata.Category] = append(itemsByCategory[destConfig.Metadata.Category], item)
	}

	for category, items := range itemsByCategory {
		resp.Categories = append(resp.Categories, DestinationsCategory{
			Name:  category,
			Items: items,
		})
	}

	c.JSON(200, resp)
}

type GetDestinationDetailsResponse struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	Name                string                 `json:"name"`
	DisplayName         string                 `json:"display_name"`
	ComponentType       string                 `json:"component_type"`
	ComponentProperties map[string]interface{} `json:"component_properties"`
	VideoUrl            string                 `json:"video_url,omitempty"`
	ThumbnailURL        string                 `json:"thumbnail_url,omitempty"`
	InitialValue        string                 `json:"initial_value,omitempty"`
}

func GetDestinationTypeDetails(c *gin.Context) {
	destType := common.DestinationType(c.Param("type"))
	destTypeConfig, err := getDestinationTypeConfig(destType)
	if err != nil {
		c.JSON(404, gin.H{
			"error": fmt.Sprintf("destination type %s not found", destType),
		})
		return
	}

	var resp GetDestinationDetailsResponse
	for _, field := range destTypeConfig.Spec.Fields {
		resp.Fields = append(resp.Fields, Field{
			Name:                field.Name,
			DisplayName:         field.DisplayName,
			ComponentType:       field.ComponentType,
			ComponentProperties: field.ComponentProps,
			VideoUrl:            field.VideoURL,
			ThumbnailURL:        field.ThumbnailURL,
			InitialValue:        field.InitialValue,
		})
	}

	c.JSON(200, resp)
}

func GetDestinations(c *gin.Context, odigosns string) {
	dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	resp := []Destination{}
	for _, dest := range dests.Items {
		secretFields, err := getDestinationSecretFields(c, odigosns, &dest)
		if err != nil {
			returnError(c, err)
			return
		}
		endpointDest := k8sDestinationToEndpointFormat(dest, secretFields)
		resp = append(resp, endpointDest)
	}

	c.JSON(200, resp)
}

func GetDestinationById(c *gin.Context, odigosns string) {
	destId := c.Param("id")
	destination, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Get(c, destId, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	secretFields, err := getDestinationSecretFields(c, odigosns, destination)
	if err != nil {
		returnError(c, err)
		return
	}
	resp := k8sDestinationToEndpointFormat(*destination, secretFields)
	c.JSON(200, resp)
}

func CreateNewDestination(c *gin.Context, odigosns string) {

	request := Destination{}
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	destType := request.Type
	destName := request.Name

	destTypeConfig, err := getDestinationTypeConfig(destType)
	if err != nil {
		returnError(c, err)
		return
	}

	errors := verifyDestinationDataScheme(destType, destTypeConfig, request.Fields)
	if len(errors) > 0 {
		returnErrors(c, errors)
		return
	}

	dataField, secretFields := transformFieldsToDataAndSecrets(destTypeConfig, request.Fields)
	generateNamePrefix := "odigos.io.dest." + string(destType) + "-"

	k8sDestination := v1alpha1.Destination{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
		},
		Spec: v1alpha1.DestinationSpec{
			Type:            destType,
			DestinationName: destName,
			Data:            dataField,
			Signals:         exportedSignalsObjectToSlice(request.ExportedSignals),
		},
	}

	createSecret := len(secretFields) > 0
	if createSecret {
		secretRef, err := createDestinationSecret(c, destType, secretFields, odigosns)
		if err != nil {
			returnError(c, err)
			return
		}
		k8sDestination.Spec.SecretRef = secretRef
	}

	dest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Create(c, &k8sDestination, metav1.CreateOptions{})
	if err != nil {
		// if we failed to create the destination, we need to rollback the secret creation
		if createSecret {
			kube.DefaultClient.CoreV1().Secrets(odigosns).Delete(c, destName, metav1.DeleteOptions{})
		}
		returnError(c, err)
		return
	}

	resp := k8sDestinationToEndpointFormat(*dest, secretFields)
	c.JSON(201, resp)
}

func UpdateExistingDestination(c *gin.Context, odigosns string) {
	destId := c.Param("id")
	request := Destination{}
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	destType := request.Type
	destName := request.Name

	destTypeConfig, err := getDestinationTypeConfig(destType)
	if err != nil {
		returnError(c, err)
		return
	}

	errors := verifyDestinationDataScheme(destType, destTypeConfig, request.Fields)
	if len(errors) > 0 {
		returnErrors(c, errors)
		return
	}

	dataFields, secretFields := transformFieldsToDataAndSecrets(destTypeConfig, request.Fields)

	// update destination
	dest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Get(c, destId, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	// handle the secret, based on the updated (which might add or remove optional secret fields),
	// we might need to create, delete or update the existing secret
	destUpdateHasSecrets := len(secretFields) > 0
	destCurrentlyHasSecrets := dest.Spec.SecretRef != nil

	if !destUpdateHasSecrets && destCurrentlyHasSecrets {
		// delete the secret if it's not needed anymore
		err := kube.DefaultClient.CoreV1().Secrets(odigosns).Delete(c, dest.Spec.SecretRef.Name, metav1.DeleteOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
		dest.Spec.SecretRef = nil
	} else if destUpdateHasSecrets && !destCurrentlyHasSecrets {
		// create the secret if it was added in this update
		secretRef, err := createDestinationSecret(c, destType, secretFields, odigosns)
		if err != nil {
			returnError(c, err)
			return
		}
		dest.Spec.SecretRef = secretRef
	} else if destUpdateHasSecrets && destCurrentlyHasSecrets {
		// update the secret in case it is modified
		secret, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Get(c, dest.Spec.SecretRef.Name, metav1.GetOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
		secret.StringData = secretFields
		_, err = kube.DefaultClient.CoreV1().Secrets(odigosns).Update(c, secret, metav1.UpdateOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
	}

	secretRef := dest.Spec.SecretRef
	var origSecret *k8s.Secret
	if secretRef != nil {
		secret, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Get(c, secretRef.Name, metav1.GetOptions{})
		if err != nil {
			returnError(c, err)
			return
		}

		// keep a copy of the object so we can rollback if needed
		origSecret = secret.DeepCopy()

		// use existing object to update the secret in k8s
		secret.StringData = secretFields
		_, err = kube.DefaultClient.CoreV1().Secrets(odigosns).Update(c, secret, metav1.UpdateOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
	}

	dest.Spec.Type = request.Type
	dest.Spec.DestinationName = destName
	dest.Spec.Data = dataFields
	dest.Spec.Signals = exportedSignalsObjectToSlice(request.ExportedSignals)

	updatedDest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Update(c, dest, metav1.UpdateOptions{})
	if err != nil {
		if origSecret != nil {
			// rollback secret, it might fail but we have nothing to do with it
			kube.DefaultClient.CoreV1().Secrets(odigosns).Update(c, origSecret, metav1.UpdateOptions{})
		}
		returnError(c, err)
		return
	}

	resp := k8sDestinationToEndpointFormat(*updatedDest, secretFields)
	c.JSON(201, resp)
}

func DeleteDestination(c *gin.Context, odigosns string) {
	destId := c.Param("id")
	currentDest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Get(c, destId, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find destination with id '" + destId + "'",
		})
		return
	}

	// delete the destination
	errDest := kube.DefaultClient.OdigosClient.Destinations(odigosns).Delete(c, destId, metav1.DeleteOptions{})

	// delete the secret if we have one
	var errSecret error
	if currentDest.Spec.SecretRef != nil && currentDest.Spec.SecretRef.Name != "" {
		secretName := currentDest.Spec.SecretRef.Name
		errSecret = kube.DefaultClient.CoreV1().Secrets(odigosns).Delete(c, secretName, metav1.DeleteOptions{})
	}

	if errDest != nil {
		returnError(c, errDest)
		return
	}

	if errSecret != nil {
		returnError(c, errDest)
		return
	}

	c.Status(204)
}

func k8sDestinationToEndpointFormat(k8sDest v1alpha1.Destination, secretFields map[string]string) Destination {
	destType := k8sDest.Spec.Type
	destName := k8sDest.Spec.DestinationName
	mergedFields := mergeDataAndSecrets(k8sDest.Spec.Data, secretFields)
	destTypeConfig := DestinationTypeConfigToCategoryItem(destinations.GetDestinationByType(string(destType)))

	return Destination{
		Id:   k8sDest.Name,
		Name: destName,
		Type: destType,
		ExportedSignals: ExportedSignals{
			Traces:  isSignalExported(k8sDest, common.TracesObservabilitySignal),
			Metrics: isSignalExported(k8sDest, common.MetricsObservabilitySignal),
			Logs:    isSignalExported(k8sDest, common.LogsObservabilitySignal),
		},
		Fields:          mergedFields,
		DestinationType: destTypeConfig,
	}
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

func isSignalExported(dest v1alpha1.Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dest.Spec.Signals {
		if s == signal {
			return true
		}
	}

	return false
}

func exportedSignalsObjectToSlice(signals ExportedSignals) []common.ObservabilitySignal {
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

func verifyDestinationDataScheme(destType common.DestinationType, destTypeConfig *destinations.Destination, data map[string]string) []error {

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

func getDestinationTypeConfig(destType common.DestinationType) (*destinations.Destination, error) {
	for _, dest := range destinations.Get() {
		if dest.Metadata.Type == destType {
			return &dest, nil
		}
	}

	return nil, fmt.Errorf("destination type %s not found", destType)
}

func transformFieldsToDataAndSecrets(destTypeConfig *destinations.Destination, fields map[string]string) (map[string]string, map[string]string) {

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

func getDestinationSecretFields(c *gin.Context, odigosns string, dest *v1alpha1.Destination) (map[string]string, error) {

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

func DestinationTypeConfigToCategoryItem(destConfig destinations.Destination) DestinationTypesCategoryItem {
	return DestinationTypesCategoryItem{
		Type:        destConfig.Metadata.Type,
		DisplayName: destConfig.Metadata.DisplayName,
		ImageUrl:    GetImageURL(destConfig.Spec.Image),
		SupportedSignals: SupportedSignals{
			Traces: ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Traces.Supported,
			},
			Metrics: ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Metrics.Supported,
			},
			Logs: ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Logs.Supported,
			},
		},
	}
}

func createDestinationSecret(ctx context.Context, destType common.DestinationType, secretFields map[string]string, odigosns string) (*k8s.LocalObjectReference, error) {
	generateNamePrefix := "odigos.io.dest." + string(destType) + "-"
	secret := k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
		},
		StringData: secretFields,
	}
	newSecret, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return &k8s.LocalObjectReference{
		Name: newSecret.Name,
	}, nil
}
