package utils

import "slices"

func IsItemIgnored(item string, ignoredItems []string) bool {
	return slices.Contains(ignoredItems, item)
}
