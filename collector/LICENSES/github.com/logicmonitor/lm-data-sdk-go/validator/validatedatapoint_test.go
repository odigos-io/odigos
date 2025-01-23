package validator

import (
	"testing"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

func TestValidateDataPoint(t *testing.T) {
	dpInput := model.DataPointInput{
		DataPointName:            "cpu",
		DataPointType:            "GAUGE",
		DataPointDescription:     "cpu",
		DataPointAggregationType: "SUM",
		Value:                    map[string]string{"123434": "33"},
	}
	msg := validateDatapoint(dpInput)
	if msg != "" {
		t.Errorf("validateDatapoint() error message= %s", msg)
		return
	}
}
func TestCheckDataPointNameValidation(t *testing.T) {

	longResName := generateLongName(266)

	type args struct {
		DataPointName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Mandatory DataPoint name",
			args: args{
				DataPointName: "",
			},
			errorMsg: "Datapoint Name is mandatory. ",
		},
		{
			name: "Empty DataPoint name",
			args: args{
				DataPointName: " tailingspacesdptest ",
			},
			errorMsg: "Datapoint Name Should not be empty or have tailing spaces. Invalid Datapoint name  tailingspacesdptest ",
		},
		{
			name: "Long DataPoint name",
			args: args{
				DataPointName: longResName,
			},
			errorMsg: "Datapoint Name size should not be greater than 128 characters. ",
		},
		{
			name: "Invalid DataPoint name",
			args: args{
				DataPointName: "cos",
			},
			errorMsg: "cos is a keyword and cannot be used as datapoint name.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkDataPointNameValidation(tt.args.DataPointName)
			if msg != tt.errorMsg {
				t.Errorf("checkDataPointNameValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckDataPointAggregationTypeValidation(t *testing.T) {
	msg := checkDataPointAggregationTypeValidation("NA")
	if msg != "The datapoint aggregation type is having invalid datapoint aggregation type: NA. " {
		t.Errorf("checkDataPointAggregationTypeValidation() unknown error = %s", msg)
		return
	}
}

func TestCheckDataPointTypeValidation(t *testing.T) {
	msg := checkDataPointTypeValidation("NA")
	if msg != "The datapoint type is having invalid dataPointType : NA. " {
		t.Errorf("checkDataPointTypeValidation() unknown error = %s", msg)
		return
	}
}

func TestCheckDataPointDescriptionValidation(t *testing.T) {
	longDesc := generateLongName(1027)
	msg := checkDataPointDescriptionValidation(longDesc)
	if msg != "Datapoint description should not be greater than 1024 characters. " {
		t.Errorf("checkDataPointDescriptionValidation() unknown error = %s", msg)
		return
	}
}

func TestCheckPercentileValue(t *testing.T) {
	msg := checkPercentileValue("percentile", "120", "GOSDK")
	if msg != "The datapoint GOSDK is not provided or having invalid percentileValue, percentileValue should be between 0-100." {
		t.Errorf("checkPercentileValue() unknown error = %s", msg)
		return
	}
}
