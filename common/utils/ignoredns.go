package utils

func arrayContainsString(arr []string, str string) bool {
	for _, elem := range arr {
		if elem == str {
			return true
		}
	}
	return false
}

func MergeDefaultIgnoreWithUserInput(userInputIgnore []string, defaultIgnored []string) []string {

	mergedList := make([]string, len(userInputIgnore))
	copy(mergedList, userInputIgnore)

	for _, ns := range defaultIgnored {
		if !arrayContainsString(mergedList, ns) {
			mergedList = append(mergedList, ns)
		}
	}

	return mergedList
}

func IsItemIgnored(item string, ignoredList []string) bool {
	for _, ignoredListItem := range ignoredList {
		if item == ignoredListItem {
			return true
		}
	}
	return false
}
