package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

const (
	REGEX_INSTANCE_NAME               = "^[a-z:A-Z0-9\\._\\-]+$"
	REGEX_INVALID_DEVICE_DISPLAY_NAME = "[*<?,;`\\n]"
)

func validateInstance(insInput model.InstanceInput) string {

	errorMsg := ""

	insID := insInput.InstanceID
	insName := insInput.InstanceName
	if insID != 0 {
		if insID > 0 {
			errorMsg += checkInstanceId(insID)
		} else {
			errorMsg += fmt.Sprintf("Instance Id %d should not be negative. ", insID)
		}
	} else {
		if insName == "" {
			errorMsg += "Either Instance Id or Instance Name is mandatory. "
		}
	}

	if insName != "" {
		errorMsg += CheckInstanceNameValidation(insName)
	}

	if insInput.InstanceDisplayName != "" {
		errorMsg += checkInsDisplayNameValidation(insInput.InstanceDisplayName)
	}
	if insInput.InstanceProperties != nil {
		errorMsg += CheckInstancePropertiesValidation(insInput.InstanceProperties)
	}

	return errorMsg
}

func CheckInstanceNameValidation(insName string) string {
	errorInstMsg := ""
	if passEmptyAndSpellCheck(insName) {
		errorInstMsg = "Instance Name Should not be empty or have tailing spaces. "
	} else if len(insName) > 255 {
		errorInstMsg = "Instance Name size should not be greater than 255 characters. "
	} else if !isValidInstanceName(insName) {
		errorInstMsg = "Invalid Instance Name " + insName + ". "
	}
	return errorInstMsg
}

func checkInsDisplayNameValidation(insDisplay string) string {
	errorInsMsg := ""
	if insDisplay != "" {
		if passEmptyAndSpellCheck(insDisplay) {
			errorInsMsg += "Instance Display Name Should not be empty or have tailing spaces. "
		}
		if len(insDisplay) > 255 {
			errorInsMsg += "Instance Display Name size should not be greater than 255 characters. "
		}
		errorInsMsg += standardChecks(insDisplay, "Instance", REGEX_INVALID_DEVICE_DISPLAY_NAME)
	}
	return errorInsMsg
}

func checkInstanceId(insID int) string {
	errorInsMsg := ""
	if !isValidId9Digit(insID) {
		errorInsMsg += "Instance Id cannot be more than 9 digit."
	}
	if isValidIdExpo(insID) {
		errorInsMsg += "Instance Id cannot be in Exponential form."
	}
	return errorInsMsg
}

func CheckInstancePropertiesValidation(insProp map[string]string) string {
	errorInsMsg := ""
	for key, value := range insProp {
		if passEmptyAndSpellCheck(key) {
			errorInsMsg += "Instance Properties Key should not be null, empty or have trailing spaces. "
		} else if len(key) > 255 {
			errorInsMsg += "Instance Properties Key should not be greater than 255 characters. "
		} else if strings.Contains(key, "##") {
			errorInsMsg += "Cannot use '##' in property name. "
		} else if strings.HasPrefix(strings.ToLower(key), "system.") || strings.HasPrefix(strings.ToLower(key), "auto.") {
			errorInsMsg += "Instance Properties should not contain system or auto properties " + key + ". "
		} else if !isValidInstanceName(key) {
			errorInsMsg += "Invalid Instance Properties key " + key + ". "
		} else if passEmptyAndSpellCheck(value) {
			errorInsMsg += "Instance Properties Value should not be null, empty or have trailing spaces. "
		} else if len(value) > 24000 {
			errorInsMsg += "Instance Properties Value should not be greater than 24000 characters. "
		} else if !isValidInstanceName(value) {
			errorInsMsg += "Invalid Instance Properties Value " + value + ". "
		}
	}
	return errorInsMsg
}

func isValidInstanceName(insName string) bool {
	match, _ := regexp.MatchString(REGEX_INSTANCE_NAME, insName)
	return match
}
