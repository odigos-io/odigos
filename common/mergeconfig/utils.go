package mergeconfig

// merge 2 optional string arrays into a new array without duplicates.
// if both are nil, returns nil.
func MergeStringArrays(a1 *[]string, a2 *[]string) *[]string {
	if a1 == nil {
		return a2
	}
	if a2 == nil {
		return a1
	}
	allMimes := map[string]struct{}{}
	for _, mime := range *a1 {
		allMimes[mime] = struct{}{}
	}
	for _, mime := range *a2 {
		allMimes[mime] = struct{}{}
	}
	mergedMimes := make([]string, 0, len(allMimes))
	for mime := range allMimes {
		mergedMimes = append(mergedMimes, mime)
	}
	return &mergedMimes
}

// merge 2 optional int64s into a new optional int64.
// if both are not nil, return the smaller value.
func MergeOptionalIntChooseLower(p1 *int64, p2 *int64) *int64 {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	if *p1 < *p2 {
		return p1
	} else {
		return p2
	}
}

// merge 2 optional bools into a new optional bool.
// if one of them is true, the result is true.
// if none is true and one of them is false, the result is false.
// if both are nil, the result is nil.
func MergeOptionalBools(p1 *bool, p2 *bool) *bool {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	if *p1 {
		return p1
	} else if *p2 {
		return p2
	} else {
		f := false
		return &f
	}
}
