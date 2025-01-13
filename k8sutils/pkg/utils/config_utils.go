package utils

func IsItemIgnored(item string, ignoredItems []string) bool {
	for _, ignoredItem := range ignoredItems {
		if item == ignoredItem {
			return true
		}
	}
	return false
}
