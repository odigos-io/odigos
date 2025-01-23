package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/logicmonitor/lm-data-sdk-go/model"
)

const (
	INVALID_REGEX    = "[^a-zA-Z $#@_0-9:&\\.\\+\n]"
	REGEX_ID_9_DIGIT = "^[0-9]{0,9}$"
	REGEX_ID_EXPO    = "^e\\^\\-?\\d*?$"
)

func ValidateAttributes(rInput model.ResourceInput, dsInput model.DatasourceInput, instInput model.InstanceInput, dpInput model.DataPointInput) string {
	errorMsg := ""
	errorMsg += validateResource(rInput)
	errorMsg += validateDatasource(dsInput)
	errorMsg += validateInstance(instInput)
	errorMsg += validateDatapoint(dpInput)
	return errorMsg
}

func passEmptyAndSpellCheck(name string) bool {
	return len(name) == 0 || strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ")
}

func standardChecks(name string, fieldName string, regex string) string {
	testStr := "##"
	if name == "" {
		return fmt.Sprintf("%s can't be null.", fieldName)
	}
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") {
		return fmt.Sprintf("Space is not allowed at start and end in %s.", fieldName)
	}
	if match, _ := regexp.MatchString(regex, name); match {
		return fmt.Sprintf("Invalid %s : %s.", fieldName, name)
	} else {
		if strings.Contains(name, testStr) {
			return fmt.Sprintf("Invalid %s : %s.", fieldName, name)
		}
	}
	return ""
}

func isValidId9Digit(id int) bool {
	match, _ := regexp.MatchString(REGEX_ID_9_DIGIT, strconv.Itoa(id))
	return match
}

func isValidIdExpo(id int) bool {
	match, _ := regexp.MatchString(REGEX_ID_EXPO, strconv.Itoa(id))
	return match
}
