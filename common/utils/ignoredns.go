package utils

func arrayContainsString(arr []string, str string) bool {
	for _, elem := range arr {
		if elem == str {
			return true
		}
	}
	return false
}

func AddSystemNamespacesToIgnored(userIgnoredNamespaces []string, systemNamespaces []string) []string {

	mergedList := make([]string, len(userIgnoredNamespaces))
	copy(mergedList, userIgnoredNamespaces)

	for _, ns := range systemNamespaces {
		if !arrayContainsString(mergedList, ns) {
			mergedList = append(mergedList, ns)
		}
	}

	return mergedList
}
