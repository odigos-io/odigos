package validator

import (
	"testing"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

func TestValidateInstance(t *testing.T) {
	insInput1 := model.InstanceInput{
		InstanceName:        "DataSDK",
		InstanceID:          0,
		InstanceDisplayName: "DataSDK",
		InstanceGroup:       "SDK",
		InstanceProperties:  map[string]string{"sdk": "test"},
	}
	insInput2 := model.InstanceInput{
		InstanceName:        "",
		InstanceID:          43,
		InstanceDisplayName: "DataSDK",
		InstanceGroup:       "SDK",
		InstanceProperties:  map[string]string{"sdk": "test"},
	}
	insInput3 := model.InstanceInput{
		InstanceName:        "",
		InstanceID:          0,
		InstanceDisplayName: "DataSDK",
		InstanceGroup:       "SDK",
		InstanceProperties:  map[string]string{"sdk": "test"},
	}
	insInput4 := model.InstanceInput{
		InstanceName:        "DataSDK",
		InstanceID:          -87,
		InstanceDisplayName: "DataSDK",
		InstanceGroup:       "SDK",
		InstanceProperties:  map[string]string{"sdk": "test"},
	}

	type args struct {
		insInput model.InstanceInput
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Instance name not present",
			args: args{
				insInput: insInput1,
			},
			errorMsg: "",
		},
		{
			name: "Instance id not present",
			args: args{
				insInput: insInput2,
			},
			errorMsg: "",
		},
		{
			name: "Both Instance name and id not present",
			args: args{
				insInput: insInput3,
			},
			errorMsg: "Either Instance Id or Instance Name is mandatory. ",
		},
		{
			name: "Negative Instance ID",
			args: args{
				insInput: insInput4,
			},
			errorMsg: "Instance Id -87 should not be negative. ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateInstance(tt.args.insInput)
			if msg != tt.errorMsg {
				t.Errorf("validateInstance() error message= %s", msg)
				return
			}
		})
	}
}

func TestCheckInstanceNameValidation(t *testing.T) {

	longInsName := generateLongName(276)

	type args struct {
		insName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Empty Instance name",
			args: args{
				insName: "",
			},
			errorMsg: "Instance Name Should not be empty or have tailing spaces. ",
		},
		{
			name: "Long Instance name",
			args: args{
				insName: longInsName,
			},
			errorMsg: "Instance Name size should not be greater than 255 characters. ",
		},
		{
			name: "Invalid Instance name",
			args: args{
				insName: "Test&#545sfd",
			},
			errorMsg: "Invalid Instance Name Test&#545sfd. ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckInstanceNameValidation(tt.args.insName)
			if msg != tt.errorMsg {
				t.Errorf("checkInstanceNameValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckInstanceDisplayNameValidation(t *testing.T) {

	longinsName := generateLongName(256)

	type args struct {
		insName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Long Instance Display name",
			args: args{
				insName: longinsName,
			},
			errorMsg: "Instance Display Name size should not be greater than 255 characters. ",
		},
		{
			name: "Tailing spaces in Instance Display name",
			args: args{
				insName: " test ",
			},
			errorMsg: "Instance Display Name Should not be empty or have tailing spaces. Space is not allowed at start and end in Instance.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkInsDisplayNameValidation(tt.args.insName)
			if msg != tt.errorMsg {
				t.Errorf("checkInsDisplayNameValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckInstanceId(t *testing.T) {
	type args struct {
		insID int
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "More than 9 digit",
			args: args{
				insID: 12345678900987,
			},
			errorMsg: "Instance Id cannot be more than 9 digit.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkInstanceId(tt.args.insID)
			if msg != tt.errorMsg {
				t.Errorf("checkInstanceId() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckInstancePropertiesValidation(t *testing.T) {

	longKey := generateLongName(256)
	longValue := generateLongName(24100)

	type args struct {
		insprop map[string]string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Empty Properties key",
			args: args{
				insprop: map[string]string{"": "value"},
			},
			errorMsg: "Instance Properties Key should not be null, empty or have trailing spaces. ",
		},
		{
			name: "Empty Properties value",
			args: args{
				insprop: map[string]string{"key": ""},
			},
			errorMsg: "Instance Properties Value should not be null, empty or have trailing spaces. ",
		},
		{
			name: "Invalid Instance Properties key",
			args: args{
				insprop: map[string]string{"Test?&*(Invalid#key": "correctvalue"},
			},
			errorMsg: "Invalid Instance Properties key Test?&*(Invalid#key. ",
		},
		{
			name: "Invalid Instance Properties value",
			args: args{
				insprop: map[string]string{"correctkey": "Test?&*(Invalid#value"},
			},
			errorMsg: "Invalid Instance Properties Value Test?&*(Invalid#value. ",
		},
		{
			name: "Long Instance ID key",
			args: args{
				insprop: map[string]string{longKey: "Test?&*(Invalid#value"},
			},
			errorMsg: "Instance Properties Key should not be greater than 255 characters. ",
		},
		{
			name: "Long Instance ID value",
			args: args{
				insprop: map[string]string{"correctkey": longValue},
			},
			errorMsg: "Instance Properties Value should not be greater than 24000 characters. ",
		},
		{
			name: "Cannot use ## in key",
			args: args{
				insprop: map[string]string{"incorr##ectkey": "valuetest"},
			},
			errorMsg: "Cannot use '##' in property name. ",
		},
		{
			name: "System and auto properties",
			args: args{
				insprop: map[string]string{"system.correctkeyproperty": "valuetest"},
			},
			errorMsg: "Instance Properties should not contain system or auto properties system.correctkeyproperty. ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckInstancePropertiesValidation(tt.args.insprop)
			if msg != tt.errorMsg {
				t.Errorf("checkInstancePropertiesValidation() unknown error = %s", msg)
				return
			}
		})
	}
}
