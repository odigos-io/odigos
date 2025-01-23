package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

const (
	REGEX_DATA_SOURCE_GROUP_NAME           = "[a-zA-Z0-9_\\- ]+$"
	REGEX_INVALID_DATA_SOURCE_DISPLAY_NAME = "[^a-zA-Z: _0-9\\(\\)\\.#\\+@<>\n]"
	REGEX_INVALID_DATA_SOURCE_NAME         = "[^a-zA-Z $#@_0-9:&\\.\\+\n]"
)

func validateDatasource(dsInput model.DatasourceInput) string {
	errorMsg := ""

	dsID := dsInput.DataSourceID
	dsName := dsInput.DataSourceName
	if dsID != 0 {
		if dsID > 0 {
			errorMsg += checkDataSourceId(dsID)
		} else {
			errorMsg += fmt.Sprintf("DataSource Id %d should not be negative.", dsID)
		}
	} else {
		if dsName == "" {
			errorMsg += "Either dataSourceId or dataSource is mandatory."
		}
	}

	if dsName != "" {
		errorMsg += CheckDataSourceNameValidation(dsName)
	}

	errorMsg += checkDSGroupValidation(dsInput.DataSourceGroup)
	errorMsg += CheckDSDisplayNameValidation(dsInput.DataSourceDisplayName)

	return errorMsg
}

func CheckDataSourceNameValidation(dsName string) string {
	errorDsMsg := ""
	if passEmptyAndSpellCheck(dsName) {
		errorDsMsg = "Datasource Name Should not be empty or have tailing spaces. "
	} else if len(dsName) > 64 {
		errorDsMsg = "Datasource Name size should not be greater than 64 characters. "
	}
	errorDsMsg += validateDSName(dsName, "Datasource Name")
	errorDsMsg += standardChecks(dsName, "Datasource", REGEX_INVALID_DATA_SOURCE_NAME)

	return errorDsMsg
}

func checkDataSourceId(dsID int) string {
	errorDsMsg := ""
	if !isValidId9Digit(dsID) {
		errorDsMsg += "DataSource Id cannot be more than 9 digit."
	}
	if isValidIdExpo(dsID) {
		errorDsMsg += "DataSource Id cannot be in Exponential form."
	}
	return errorDsMsg
}

func CheckDSDisplayNameValidation(dsDisplay string) string {
	errorDsMsg := ""
	if dsDisplay != "" {
		if passEmptyAndSpellCheck(dsDisplay) {
			errorDsMsg += "Datasource Display Name Should not be empty or have tailing spaces. "
		}
		if len(dsDisplay) > 64 {
			errorDsMsg += "Datasource Display Name size should not be greater than 64 characters. "
		}
		errorDsMsg += validateDSName(dsDisplay, "Datasource Display Name")
		errorDsMsg += standardChecks(dsDisplay, "Datasource", REGEX_INVALID_DATA_SOURCE_DISPLAY_NAME)
	}
	return errorDsMsg
}

func checkDSGroupValidation(dsGroup string) string {
	errorDsMsg := ""
	if dsGroup != "" {
		if passEmptyAndSpellCheck(dsGroup) {
			errorDsMsg += "Datasource Group Name Should not be empty or have tailing spaces."
		}
		if len(dsGroup) < 2 || len(dsGroup) > 128 {
			errorDsMsg += "Datasource Group Name size should not be less than 2 or greater than 128 characters."
		}
		if !isValidDataSourceGroupName(dsGroup) {
			errorDsMsg += "Invalid Datasource Group Name: " + dsGroup
		}
	}
	return errorDsMsg
}

func isValidDataSourceGroupName(dsGroup string) bool {
	match, _ := regexp.MatchString(REGEX_DATA_SOURCE_GROUP_NAME, dsGroup)
	return match
}

func validateDSName(dsName string, fieldName string) string {
	errorDsMsg := ""
	if strings.Contains(dsName, "-") {
		if strings.Index(dsName, "-") == len(dsName)-1 {
			if len(dsName) == 1 {
				errorDsMsg = fmt.Sprintf("%s cannot be single \"-\".", fieldName)
			}
		} else {
			errorDsMsg = fmt.Sprintf("Support \"-\" for %s when its the last char.", fieldName)
		}
	}
	return errorDsMsg
}
