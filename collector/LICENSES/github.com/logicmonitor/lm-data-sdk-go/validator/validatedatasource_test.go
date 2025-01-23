package validator

import (
	"testing"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

func TestValidateDatasource(t *testing.T) {
	dsInput1 := model.DatasourceInput{
		DataSourceName:        "GoDataSDK",
		DataSourceDisplayName: "GoDataSDK",
		DataSourceGroup:       "SDK",
		DataSourceID:          0,
	}
	dsInput2 := model.DatasourceInput{
		DataSourceName:        "",
		DataSourceDisplayName: "GoDataSDK",
		DataSourceGroup:       "SDK",
		DataSourceID:          123,
	}
	dsInput3 := model.DatasourceInput{
		DataSourceDisplayName: "GoDataSDK",
		DataSourceGroup:       "SDK",
	}
	dsInput4 := model.DatasourceInput{
		DataSourceDisplayName: "GoDataSDK",
		DataSourceGroup:       "SDK",
		DataSourceID:          -78,
	}

	type args struct {
		dsInput model.DatasourceInput
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Datasource name not present",
			args: args{
				dsInput: dsInput1,
			},
			errorMsg: "",
		},
		{
			name: "Datasource id not present",
			args: args{
				dsInput: dsInput2,
			},
			errorMsg: "",
		},
		{
			name: "Both Datasource name and id not present",
			args: args{
				dsInput: dsInput3,
			},
			errorMsg: "Either dataSourceId or dataSource is mandatory.",
		},
		{
			name: "Negative Datasource ID",
			args: args{
				dsInput: dsInput4,
			},
			errorMsg: "DataSource Id -78 should not be negative.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateDatasource(tt.args.dsInput)
			if msg != tt.errorMsg {
				t.Errorf("validateDatasource() error message= %s", msg)
				return
			}
		})
	}
}

func TestCheckDataSourceNameValidation(t *testing.T) {

	longDsName := generateLongName(67)

	type args struct {
		dsName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Empty Datasource name",
			args: args{
				dsName: "",
			},
			errorMsg: "Datasource Name Should not be empty or have tailing spaces. Datasource can't be null.",
		},
		{
			name: "Long Datasource name",
			args: args{
				dsName: longDsName,
			},
			errorMsg: "Datasource Name size should not be greater than 64 characters. ",
		},
		{
			name: "Support hyphen only as last char",
			args: args{
				dsName: "Test-name",
			},
			errorMsg: "Support \"-\" for Datasource Name when its the last char.Invalid Datasource : Test-name.",
		},
		{
			name: "Single hyphen",
			args: args{
				dsName: "-",
			},
			errorMsg: "Datasource Name cannot be single \"-\".Invalid Datasource : -.",
		},
		{
			name: "No space at start or end",
			args: args{
				dsName: " testspacevalidation ",
			},
			errorMsg: "Datasource Name Should not be empty or have tailing spaces. Space is not allowed at start and end in Datasource.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckDataSourceNameValidation(tt.args.dsName)
			if msg != tt.errorMsg {
				t.Errorf("checkDataSourceNameValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckDataSourceDisplayNameValidation(t *testing.T) {

	longDsName := generateLongName(67)

	type args struct {
		dsName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Long Datasource name",
			args: args{
				dsName: longDsName,
			},
			errorMsg: "Datasource Display Name size should not be greater than 64 characters. ",
		},
		{
			name: "Support hyphen only as last char",
			args: args{
				dsName: "Test-name",
			},
			errorMsg: "Support \"-\" for Datasource Display Name when its the last char.Invalid Datasource : Test-name.",
		},
		{
			name: "Single hyphen",
			args: args{
				dsName: "-",
			},
			errorMsg: "Datasource Display Name cannot be single \"-\".Invalid Datasource : -.",
		},
		{
			name: "No space at start or end",
			args: args{
				dsName: " testspacevalidation ",
			},
			errorMsg: "Datasource Display Name Should not be empty or have tailing spaces. Space is not allowed at start and end in Datasource.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CheckDSDisplayNameValidation(tt.args.dsName)
			if msg != tt.errorMsg {
				t.Errorf("checkDSDisplayNameValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckDataSourceGroupValidation(t *testing.T) {

	longDsGroupName := generateLongName(130)

	type args struct {
		dsName string
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "Empty Datasource Group name",
			args: args{
				dsName: " test ",
			},
			errorMsg: "Datasource Group Name Should not be empty or have tailing spaces.",
		},
		{
			name: "Long Datasource name",
			args: args{
				dsName: longDsGroupName,
			},
			errorMsg: "Datasource Group Name size should not be less than 2 or greater than 128 characters.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkDSGroupValidation(tt.args.dsName)
			if msg != tt.errorMsg {
				t.Errorf("checkDSGroupValidation() unknown error = %s", msg)
				return
			}
		})
	}
}

func TestCheckDataSourceId(t *testing.T) {
	type args struct {
		dsName int
	}

	tests := []struct {
		name     string
		args     args
		errorMsg string
	}{
		{
			name: "More than 9 digit",
			args: args{
				dsName: 12345678900987,
			},
			errorMsg: "DataSource Id cannot be more than 9 digit.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkDataSourceId(tt.args.dsName)
			if msg != tt.errorMsg {
				t.Errorf("checkDataSourceId() unknown error = %s", msg)
				return
			}
		})
	}
}
