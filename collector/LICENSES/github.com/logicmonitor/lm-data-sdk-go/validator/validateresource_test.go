package validator

import (
	"math/rand"
	"testing"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

func TestValidateResource(t *testing.T) {
	rInput := model.ResourceInput{
		ResourceName:        "TestResourceName",
		ResourceDescription: "TestDescription",
		ResourceID:          map[string]string{"system.displayname": "TestResourceName"},
		ResourceProperties:  map[string]string{"newkey": "newvalue"},
		IsCreate:            false,
	}
	msg := validateResource(rInput)
	if msg != "" {
		t.Errorf("validateResource() error message= %s", msg)
		return
	}
}

func TestCheckResourceNameValidation(t *testing.T) {

	longResName := generateLongName(266)

	type args struct {
		isCreate     bool
		resourceName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Mandatory resource name if isCreate true",
			args: args{
				isCreate:     true,
				resourceName: "",
			},
			errorMsg: "Resource name is mandatory. ",
		},
		{
			name: "Empty resource name",
			args: args{
				isCreate:     false,
				resourceName: "",
			},
			errorMsg: "Resource Name Should not be empty or have tailing spaces. ",
		},
		{
			name: "Long Resource name",
			args: args{
				isCreate:     false,
				resourceName: longResName,
			},
			errorMsg: "Resource Name size should not be greater than 255 characters. ",
		},
		{
			name: "Invalid Resource name",
			args: args{
				isCreate:     false,
				resourceName: "Test?&*(Invalid#423name",
			},
			errorMsg: "Invalid Resource Name Test?&*(Invalid#423name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckResourceNameValidation(tt.args.isCreate, tt.args.resourceName)
			if msg != tt.errorMsg {
				t.Errorf("checkResourceNameValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckResourceDescriptionValidation(t *testing.T) {

	longDescription := generateLongName(65537)

	type args struct {
		resourceDesc string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Long Description",
			args: args{
				resourceDesc: longDescription,
			},
			errorMsg: "Resource Description Size should not be greater than 65535 characters. ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkResourceDescriptionValidation(tt.args.resourceDesc)
			if msg != tt.errorMsg {
				t.Errorf("checkResourceDescriptionValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckResourceIDValidation(t *testing.T) {

	longKey := generateLongName(256)
	longValue := generateLongName(24100)

	type args struct {
		resid map[string]string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Mandatory Resource ID",
			args: args{
				resid: map[string]string{},
			},
			errorMsg: "Resource IDs is mandatory. ",
		},
		{
			name: "Empty key",
			args: args{
				resid: map[string]string{"": "value"},
			},
			errorMsg: "Resource Id Key should not be null, empty or have trailing spaces. ",
		},
		{
			name: "Empty value",
			args: args{
				resid: map[string]string{"key": ""},
			},
			errorMsg: "Resource Id Value should not be null, empty or have trailing spaces. ",
		},
		{
			name: "Invalid Resource ID key",
			args: args{
				resid: map[string]string{"Test?&*(Invalid#key": "correctvalue"},
			},
			errorMsg: "Invalid Resource ID key Test?&*(Invalid#key. ",
		},
		{
			name: "Invalid Resource ID value",
			args: args{
				resid: map[string]string{"correctkey": "Test?&*(Invalid#value"},
			},
			errorMsg: "Invalid Resource ID Value Test?&*(Invalid#value. ",
		},
		{
			name: "Long Resource ID key",
			args: args{
				resid: map[string]string{longKey: "Test?&*(Invalid#value"},
			},
			errorMsg: "Resource Id Key should not be greater than 255 characters. ",
		},
		{
			name: "Long Resource ID value",
			args: args{
				resid: map[string]string{"correctkey": longValue},
			},
			errorMsg: "Resource Id Value should not be greater than 24000 characters. ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckResourceIDValidation(tt.args.resid)
			if msg != tt.errorMsg {
				t.Errorf("checkResourceIDValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckResourcePropertiesValidation(t *testing.T) {

	longKey := generateLongName(256)
	longValue := generateLongName(24100)

	type args struct {
		resprop map[string]string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Empty Properties key",
			args: args{
				resprop: map[string]string{"": "value"},
			},
			errorMsg: "Resource Properties Key should not be null, empty or have trailing spaces. ",
		},
		{
			name: "Empty Properties value",
			args: args{
				resprop: map[string]string{"key": ""},
			},
			errorMsg: "Resource Properties Value should not be null, empty or have trailing spaces. ",
		},
		{
			name: "Invalid Resource Properties key",
			args: args{
				resprop: map[string]string{"Test?&*(Invalid#key": "correctvalue"},
			},
			errorMsg: "Invalid Resource Properties key Test?&*(Invalid#key. ",
		},
		{
			name: "Invalid Resource Properties value",
			args: args{
				resprop: map[string]string{"correctkey": "Test?&*(Invalid#value"},
			},
			errorMsg: "Invalid Resource Properties Value Test?&*(Invalid#value. ",
		},
		{
			name: "Long Resource ID key",
			args: args{
				resprop: map[string]string{longKey: "Test?&*(Invalid#value"},
			},
			errorMsg: "Resource Properties Key should not be greater than 255 characters. ",
		},
		{
			name: "Long Resource ID value",
			args: args{
				resprop: map[string]string{"correctkey": longValue},
			},
			errorMsg: "Resource Properties Value should not be greater than 24000 characters. ",
		},
		{
			name: "Cannot use ## in key",
			args: args{
				resprop: map[string]string{"incorr##ectkey": "valuetest"},
			},
			errorMsg: "Cannot use '##' in property name. ",
		},
		{
			name: "System and auto properties",
			args: args{
				resprop: map[string]string{"system.correctkeyproperty": "valuetest"},
			},
			errorMsg: "Resource Properties should not contain system or auto properties : system.correctkeyproperty. ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckResourcePropertiesValidation(tt.args.resprop)
			if msg != tt.errorMsg {
				t.Errorf("checkResourcePropertiesValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func generateLongName(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
