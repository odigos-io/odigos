package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

const REGEX_DATA_POINT = "^[a-zA-Z_0-9]+$"

func validateDatapoint(dpInput model.DataPointInput) string {
	errorMsg := ""
	var dpValue string
	errorMsg += checkDataPointNameValidation(dpInput.DataPointName)
	if dpInput.DataPointDescription != "" {
		errorMsg += checkDataPointDescriptionValidation(dpInput.DataPointDescription)
	}
	if dpInput.DataPointAggregationType != "" {
		errorMsg += checkDataPointAggregationTypeValidation(dpInput.DataPointAggregationType)
	}
	if dpInput.DataPointType != "" {
		errorMsg += checkDataPointTypeValidation(dpInput.DataPointType)
	}
	for _, value := range dpInput.Value {
		dpValue = value
	}
	errorMsg += checkPercentileValue(dpInput.DataPointAggregationType, dpValue, dpInput.DataPointName)
	return errorMsg
}

func checkDataPointNameValidation(dpName string) string {
	errorDpMsg := ""
	if dpName == "" {
		errorDpMsg += "Datapoint Name is mandatory. "
	} else {
		if passEmptyAndSpellCheck(dpName) {
			errorDpMsg += "Datapoint Name Should not be empty or have tailing spaces. "
		}
		if len(dpName) > 128 {
			errorDpMsg += "Datapoint Name size should not be greater than 128 characters. "
		}
		errorDpMsg += validateDPName(dpName)
	}

	return errorDpMsg
}

func checkDataPointAggregationTypeValidation(aggType string) string {
	errorDpMsg := ""
	flag := false
	validAggregationDatapointType := []string{"none", "avg", "sum", "percentile"}
	for _, a := range validAggregationDatapointType {
		if strings.EqualFold(a, strings.ToLower(aggType)) {
			flag = true
		}
	}
	if !flag {
		errorDpMsg += fmt.Sprintf("The datapoint aggregation type is having invalid datapoint aggregation type: %s. ", aggType)
	}
	return errorDpMsg
}

func checkDataPointDescriptionValidation(desc string) string {
	errorDpMsg := ""
	if len(desc) > 1024 {
		errorDpMsg += "Datapoint description should not be greater than 1024 characters. "
	}
	return errorDpMsg
}

func checkDataPointTypeValidation(dpType string) string {
	errorDpMsg := ""
	flag := false
	validDatapointType := []string{"gauge", "counter", "derive"}
	for _, a := range validDatapointType {
		if strings.EqualFold(dpType, a) {
			flag = true
		}
	}
	if !flag {
		errorDpMsg += fmt.Sprintf("The datapoint type is having invalid dataPointType : %s. ", dpType)
	}
	return errorDpMsg
}

func checkPercentileValue(aggType, val, name string) string {
	errorDpMsg := ""
	intval, _ := strconv.Atoi(val)
	if strings.EqualFold(aggType, "percentile") {
		if intval <= 0 || intval >= 100 {
			errorDpMsg = fmt.Sprintf("The datapoint %s is not provided or having invalid percentileValue, percentileValue should be between 0-100.", name)
		}
	}
	return errorDpMsg
}

func validateDPName(name string) string {
	errorDpMsg := ""
	invalidDataPointNameSet := []string{
		"SIN",
		"COS",
		"LOG",
		"EXP",
		"FLOOR",
		"CEIL",
		"ROUND",
		"POW",
		"ABS",
		"SQRT",
		"RANDOM",
		"LT",
		"LE",
		"GT",
		"GE",
		"EQ",
		"NE",
		"IF",
		"MIN",
		"MAX",
		"LIMIT",
		"DUP",
		"EXC",
		"POP",
		"UN",
		"UNKN",
		"NOW",
		"TIME",
		"PI",
		"E",
		"AND",
		"OR",
		"XOR",
		"INF",
		"NEGINF",
		"STEP",
		"YEAR",
		"MONTH",
		"DATE",
		"HOUR",
		"MINUTE",
		"SECOND",
		"WEEK",
		"SIGN",
		"RND",
		"SUM2",
		"AVG2",
		"PERCENT",
		"RAWPERCENTILE",
		"IN",
		"NANTOZERO",
		"MIN2",
		"MAX2",
	}

	flag := false
	match, _ := regexp.MatchString(REGEX_DATA_POINT, name)
	if !match {
		errorDpMsg += "Invalid Datapoint name " + name
	}
	for _, a := range invalidDataPointNameSet {
		if strings.EqualFold(name, a) {
			flag = true
		}
	}
	if flag {
		errorDpMsg += fmt.Sprintf("%s is a keyword and cannot be used as datapoint name.", name)
	}
	return errorDpMsg
}
