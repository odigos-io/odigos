package odigosconfiguration

func mergeIgnoredItemLists(l1 []string, l2 []string) []string {

	merged := map[string]struct{}{}

	for _, i := range l1 {
		merged[i] = struct{}{}
	}
	for _, i := range l2 {
		merged[i] = struct{}{}
	}

	mergedList := make([]string, 0, len(merged))
	for i := range merged {
		mergedList = append(mergedList, i)
	}

	return mergedList
}

func removeItemFromList(list []string, itemToRemove string) []string {
	result := make([]string, 0, len(list))
	for _, item := range list {
		if item != itemToRemove {
			result = append(result, item)
		}
	}
	return result
}
