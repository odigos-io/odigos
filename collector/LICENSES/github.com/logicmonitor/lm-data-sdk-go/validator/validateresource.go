package validator

import (
	"regexp"
	"strings"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

const (
	VALID_REGEX_RESOURCE_NAME = "^[a-z:A-Z0-9\\._\\-]+$"
)

func validateResource(resInput model.ResourceInput) string {
	errorMsg := ""
	if resInput.ResourceName != "" {
		errorMsg += CheckResourceNameValidation(resInput.IsCreate, resInput.ResourceName)
	}
	if resInput.ResourceDescription != "" {
		errorMsg += checkResourceDescriptionValidation(resInput.ResourceName)
	}
	errorMsg += CheckResourceIDValidation(resInput.ResourceID)
	if resInput.ResourceProperties != nil {
		errorMsg += CheckResourcePropertiesValidation(resInput.ResourceProperties)
	}
	return errorMsg
}

func CheckResourceNameValidation(isCreate bool, resName string) string {
	errorResMsg := ""
	if isCreate && resName == "" {
		errorResMsg = "Resource name is mandatory. "
	} else {
		if passEmptyAndSpellCheck(resName) {
			errorResMsg = "Resource Name Should not be empty or have tailing spaces. "
		} else if len(resName) > 255 {
			errorResMsg = "Resource Name size should not be greater than 255 characters. "
		} else if isInvalidResourceName(resName) {
			errorResMsg = "Invalid Resource Name " + resName
		}
	}
	return errorResMsg
}

func checkResourceDescriptionValidation(resDesc string) string {
	errorResMsg := ""
	if len(resDesc) > 65535 {
		errorResMsg = "Resource Description Size should not be greater than 65535 characters. "
	}
	return errorResMsg
}

func CheckResourceIDValidation(resId map[string]string) string {
	errorResMsg := ""
	if resId == nil || len(resId) == 0 {
		errorResMsg = "Resource IDs is mandatory. "
	} else {
		for key, value := range resId {
			if passEmptyAndSpellCheck(key) {
				errorResMsg = "Resource Id Key should not be null, empty or have trailing spaces. "
			} else if len(key) > 255 {
				errorResMsg = "Resource Id Key should not be greater than 255 characters. "
			} else if isInvalidResourceName(key) {
				errorResMsg = "Invalid Resource ID key " + key + ". "
			} else if passEmptyAndSpellCheck(value) {
				errorResMsg = "Resource Id Value should not be null, empty or have trailing spaces. "
			} else if len(value) > 24000 {
				errorResMsg = "Resource Id Value should not be greater than 24000 characters. "
			} else if isInvalidResourceName(value) {
				errorResMsg = "Invalid Resource ID Value " + value + ". "
			}
		}
	}
	return errorResMsg
}

func CheckResourcePropertiesValidation(resProp map[string]string) string {
	errorResMsg := ""
	for key, value := range resProp {
		if passEmptyAndSpellCheck(key) {
			errorResMsg += "Resource Properties Key should not be null, empty or have trailing spaces. "
		} else if len(key) > 255 {
			errorResMsg += "Resource Properties Key should not be greater than 255 characters. "
		} else if strings.Contains(key, "##") {
			errorResMsg += "Cannot use '##' in property name. "
		} else if strings.HasPrefix(strings.ToLower(key), "system.") || strings.HasPrefix(strings.ToLower(key), "auto.") {
			errorResMsg += "Resource Properties should not contain system or auto properties : " + key + ". "
		} else if isInvalidResourceName(key) {
			errorResMsg += "Invalid Resource Properties key " + key + ". "
		} else if passEmptyAndSpellCheck(value) {
			errorResMsg += "Resource Properties Value should not be null, empty or have trailing spaces. "
		} else if len(value) > 24000 {
			errorResMsg += "Resource Properties Value should not be greater than 24000 characters. "
		} else if isInvalidResourceName(value) {
			errorResMsg += "Invalid Resource Properties Value " + value + ". "
		}
	}
	return errorResMsg
}

func isInvalidResourceName(resName string) bool {
	match, _ := regexp.MatchString(VALID_REGEX_RESOURCE_NAME, resName)
	return !match
}
