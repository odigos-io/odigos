package endpoints

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/frontend/destinations"
	"github.com/keyval-dev/odigos/frontend/kube"
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
	Name            string                 `json:"name"`
	Type            common.DestinationType `json:"type"`
	ExportedSignals ExportedSignals        `json:"signals"`
	Data            map[string]string      `json:"data"`
}

func GetDestinationTypes(c *gin.Context) {
	var resp GetDestinationTypesResponse
	itemsByCategory := make(map[string][]DestinationTypesCategoryItem)
	for _, dest := range destinations.Get() {
		item := DestinationTypesCategoryItem{
			Type:        dest.Metadata.Type,
			DisplayName: dest.Metadata.DisplayName,
			ImageUrl:    GetImageURL(dest.Spec.Image),
			SupportedSignals: SupportedSignals{
				Traces: ObservabilitySignalSupport{
					Supported: dest.Spec.Signals.Traces.Supported,
				},
				Metrics: ObservabilitySignalSupport{
					Supported: dest.Spec.Signals.Metrics.Supported,
				},
				Logs: ObservabilitySignalSupport{
					Supported: dest.Spec.Signals.Logs.Supported,
				},
			},
		}

		itemsByCategory[dest.Metadata.Category] = append(itemsByCategory[dest.Metadata.Category], item)
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
	VideoUrl            string                 `json:"video_url"`
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
		})
	}

	c.JSON(200, resp)
	return
}

func GetDestinations(c *gin.Context, odigosns string) {
	dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	var resp []Destination
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

func GetDestinationByName(c *gin.Context, odigosns string) {
	destName := c.Param("name")
	destination, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Get(c, destName, metav1.GetOptions{})
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

	errors := verifyDestinationDataScheme(destType, destTypeConfig, request.Data)
	if len(errors) > 0 {
		returnErrors(c, errors)
		return
	}

	dataField, secretFields := transformFieldsToDataAndSecrets(destTypeConfig, request.Data)

	destSpec := v1alpha1.DestinationSpec{
		Type:    destType,
		Data:    dataField,
		Signals: exportedSignalsObjectToSlice(request.ExportedSignals),
	}

	if len(secretFields) > 0 {
		destSpec.SecretRef = &k8s.LocalObjectReference{
			Name: destName,
		}
		secret := k8s.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: destName,
			},
			StringData: secretFields,
		}
		_, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Create(c, &secret, metav1.CreateOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
	}

	k8sDestination := v1alpha1.Destination{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: destName,
		},
		Spec:   destSpec,
		Status: v1alpha1.DestinationStatus{},
	}
	dest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Create(c, &k8sDestination, metav1.CreateOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	resp := k8sDestinationToEndpointFormat(*dest, secretFields)
	c.JSON(201, resp)
}

func UpdateExistingDestination(c *gin.Context, odigosns string) {
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

	errors := verifyDestinationDataScheme(destType, destTypeConfig, request.Data)
	if len(errors) > 0 {
		returnErrors(c, errors)
		return
	}

	dataFields, secretFields := transformFieldsToDataAndSecrets(destTypeConfig, request.Data)

	// update destination
	dest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Get(c, destName, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	secretRef := dest.Spec.SecretRef
	if secretRef != nil {
		secret, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Get(c, secretRef.Name, metav1.GetOptions{})
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

	dest.Spec.Type = request.Type
	dest.Spec.Data = dataFields
	dest.Spec.Signals = exportedSignalsObjectToSlice(request.ExportedSignals)

	updatedDest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Update(c, dest, metav1.UpdateOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	resp := k8sDestinationToEndpointFormat(*updatedDest, secretFields)
	c.JSON(201, resp)
}

func DeleteDestination(c *gin.Context, odigosns string) {
	destName := c.Param("name")
	errDest := kube.DefaultClient.OdigosClient.Destinations(odigosns).Delete(c, destName, metav1.DeleteOptions{})
	errSecret := kube.DefaultClient.CoreV1().Secrets(odigosns).Delete(c, destName, metav1.DeleteOptions{})

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
	destName := k8sDest.Name
	mergedFields := mergeDataAndSecrets(k8sDest.Spec.Data, secretFields)

	return Destination{
		Name: destName,
		Type: destType,
		ExportedSignals: ExportedSignals{
			Traces:  isSignalExported(k8sDest, common.TracesObservabilitySignal),
			Metrics: isSignalExported(k8sDest, common.MetricsObservabilitySignal),
			Logs:    isSignalExported(k8sDest, common.LogsObservabilitySignal),
		},
		Data: mergedFields,
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
