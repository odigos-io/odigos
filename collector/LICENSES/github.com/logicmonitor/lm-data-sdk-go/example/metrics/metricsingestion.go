package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/logicmonitor/lm-data-sdk-go/api/metrics"
	"github.com/logicmonitor/lm-data-sdk-go/model"
)

func main() {
	options := []metrics.Option{
		metrics.WithMetricBatchingInterval(3 * time.Second),
		metrics.WithRateLimit(2),
	}

	lmMetric, err := metrics.NewLMMetricIngest(context.Background(), options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when initializing metric client: %v\n", err)
		return
	}

	rInput, dsInput, insInput, dpInput := createInput1()
	_, err = lmMetric.SendMetrics(context.Background(), rInput, dsInput, insInput, dpInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when sending 1st metric: %v\n", err)
	}
	time.Sleep(1 * time.Second)

	rInput1, dsInput1, insInput1, dpInput1 := createInput2()
	_, err = lmMetric.SendMetrics(context.Background(), rInput1, dsInput1, insInput1, dpInput1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when sending 2nd metric: %v\n", err)
	}
	time.Sleep(2 * time.Second)

	rInput2, dsInput2, insInput2, dpInput2 := createInput3()
	_, err = lmMetric.SendMetrics(context.Background(), rInput2, dsInput2, insInput2, dpInput2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when sending 3rd metric: %v\n", err)
	}

	resName := "example-cart-service"
	resProp := map[string]string{"propkey": "updatedprop"}
	rId := map[string]string{"system.displayname": "example-cart-service"}
	insProp := map[string]string{"propkey": "updatedprop"}
	dsName := "TestDataSource"
	dsDisplayName := "TestDisplayName"
	insName := "DataSDK"
	patch := true

	_, err = lmMetric.UpdateInstanceProperties(rId, insProp, dsName, dsDisplayName, insName, patch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when updating instance properties: %v\n", err)
	}

	_, err = lmMetric.UpdateResourceProperties(resName, rId, resProp, patch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when updating resource properties: %v\n", err)
	}
	time.Sleep(10 * time.Second)
}

func createInput1() (model.ResourceInput, model.DatasourceInput, model.InstanceInput, model.DataPointInput) {
	// fill the values
	rInput := model.ResourceInput{
		ResourceName: "example-payment-service",
		ResourceID:   map[string]string{"system.displayname": "example-payment-service"},
	}

	dsInput := model.DatasourceInput{
		DataSourceName:        "GoSDK",
		DataSourceDisplayName: "GoSDK",
		DataSourceGroup:       "Sdk",
	}

	insInput := model.InstanceInput{
		InstanceName:       "DataSDK",
		InstanceProperties: map[string]string{"test": "datasdk"},
	}

	dpInput := model.DataPointInput{
		DataPointName:            "cpu",
		DataPointType:            "COUNTER",
		DataPointAggregationType: "SUM",
		Value:                    map[string]string{fmt.Sprintf("%d", time.Now().Unix()): "124"},
	}
	return rInput, dsInput, insInput, dpInput
}

func createInput2() (model.ResourceInput, model.DatasourceInput, model.InstanceInput, model.DataPointInput) {
	// fill the values
	rInput := model.ResourceInput{
		ResourceName: "example-checkout-service",
		ResourceID:   map[string]string{"system.displayname": "example-checkout-service"},
		IsCreate:     true,
	}

	dsInput := model.DatasourceInput{
		DataSourceName:        "JavaSDK",
		DataSourceDisplayName: "JavaSDK",
		DataSourceGroup:       "Sdk",
	}

	insInput := model.InstanceInput{
		InstanceName:       "TelemetrySDK",
		InstanceProperties: map[string]string{"test": "telemetrysdk"},
	}

	dpInput := model.DataPointInput{
		DataPointName:            "cpu",
		DataPointType:            "GAUGE",
		DataPointAggregationType: "SUM",
		Value:                    map[string]string{fmt.Sprintf("%d", time.Now().Unix()): "124"},
	}
	return rInput, dsInput, insInput, dpInput
}

func createInput3() (model.ResourceInput, model.DatasourceInput, model.InstanceInput, model.DataPointInput) {
	// fill the values
	rInput := model.ResourceInput{
		ResourceName: "example-cart-service",
		ResourceID:   map[string]string{"system.displayname": "example-cart-service"},
		IsCreate:     true,
	}

	dsInput := model.DatasourceInput{
		DataSourceName:        "GoSDK",
		DataSourceDisplayName: "GoSDK",
		DataSourceGroup:       "Sdk",
	}

	insInput := model.InstanceInput{
		InstanceName:       "TelemetrySDK",
		InstanceProperties: map[string]string{"test": "telemetrysdk"},
	}

	dpInput := model.DataPointInput{
		DataPointName:            "memory",
		DataPointType:            "GAUGE",
		DataPointAggregationType: "SUM",
		Value:                    map[string]string{fmt.Sprintf("%d", time.Now().Unix()): "14"},
	}
	return rInput, dsInput, insInput, dpInput
}
