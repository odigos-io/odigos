package flamegraph

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LocationInfo holds mapping index and address for a location (for backend symbolization).
type LocationInfo struct {
	MappingIndex int
	Address      uint64
}

// MappingInfo holds filename and build_id from the OTLP mapping table (for debuginfod lookup).
type MappingInfo struct {
	Filename string
	BuildID  string
}

// ParsedChunk holds extracted samples and name lookup from one OTLP/JSON chunk.
type ParsedChunk struct {
	Names         map[int]string            // location index -> symbol name (from dictionary)
	Samples       []Sample                  // each sample: stack (root-first) and value
	LocationInfos map[int]LocationInfo      // location index -> (mappingIndex, address); for symbolization
	MappingInfos  map[int]MappingInfo       // mapping index -> (filename, build_id)
}

// Sample is one profile sample: stack of frame names (root first) and value (e.g. count).
// LocIndices are the location table indices (root-first) so we can re-resolve names with a symbolizer.
type Sample struct {
	Stack     []string
	Value     int64
	LocIndices []int // same length as Stack; location table indices for each frame
}

// stackTableMap is stack_index -> location_indices (root-first). Used for v1development format where Sample.stack_index references ProfilesDictionary.stack_table.
type stackTableMap map[int][]int

// ParseOTLPChunk parses one OTLP/JSON profile chunk (as produced by pprofile.JSONMarshaler)
// and returns samples with resolved stack names. Handles camelCase and snake_case keys.
// Also extracts LocationInfos and MappingInfos for backend symbolization (mapping+address → name).
// Supports both OTEP (locations_start_index/locations_length) and v1development (stack_index → stack_table → location_indices) formats.
func ParseOTLPChunk(data []byte) (*ParsedChunk, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	names := make(map[int]string)
	locationInfos := make(map[int]LocationInfo)
	mappingInfos := make(map[int]MappingInfo)
	var stackTable stackTableMap
	if dict := getKey(raw, "dictionary", "Dictionary"); dict != nil {
		if dm, ok := dict.(map[string]interface{}); ok {
			stackTable = extractStackTable(dm)
		}
	}
	var samples []Sample
	extractSamplesAndNames(raw, &samples, names, locationInfos, mappingInfos, stackTable)
	return &ParsedChunk{
		Names:         names,
		Samples:       samples,
		LocationInfos: locationInfos,
		MappingInfos:  mappingInfos,
	}, nil
}

// ParsedChunkHasDictionary returns true if the chunk had a non-empty dictionary (names or location/mapping tables).
// Used to pick a reference chunk for resolving names when other chunks have empty dictionary.
func ParsedChunkHasDictionary(p *ParsedChunk) bool {
	if p == nil {
		return false
	}
	if len(p.Names) > 0 {
		return true
	}
	if len(p.LocationInfos) > 0 || len(p.MappingInfos) > 0 {
		return true
	}
	return false
}

func getKey(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	}
	return 0
}

func extractSamplesAndNames(obj interface{}, samples *[]Sample, names map[int]string, locationInfos map[int]LocationInfo, mappingInfos map[int]MappingInfo, stackTable stackTableMap) {
	if obj == nil {
		return
	}
	m, ok := obj.(map[string]interface{})
	if !ok {
		return
	}
	if data := getKey(m, "data", "Data"); data != nil {
		if dm, ok := data.(map[string]interface{}); ok {
			extractSamplesAndNames(dm, samples, names, locationInfos, mappingInfos, stackTable)
			return
		}
	}
	// Fill names and location/mapping tables from dictionary.
	if dict := getKey(m, "dictionary", "Dictionary"); dict != nil {
		if dm, ok := dict.(map[string]interface{}); ok {
			extractNamesFromDictionary(dm, names)
			if locationInfos != nil && mappingInfos != nil {
				extractLocationAndMappingTables(dm, locationInfos, mappingInfos)
			}
		}
	}
	if schema := getKey(m, "schema", "Schema"); schema != nil {
		if sm, ok := schema.(map[string]interface{}); ok {
			if dict := getKey(sm, "dictionary", "Dictionary"); dict != nil {
				if dm, ok := dict.(map[string]interface{}); ok {
					extractNamesFromDictionary(dm, names)
					if locationInfos != nil && mappingInfos != nil {
						extractLocationAndMappingTables(dm, locationInfos, mappingInfos)
					}
				}
			}
		}
	}
	extractNamesFromObject(m, names)

	if rps := getKey(m, "resourceProfiles", "ResourceProfiles", "resource_profiles"); rps != nil {
		if arr, ok := rps.([]interface{}); ok {
			for _, rp := range arr {
				if rpm, ok := rp.(map[string]interface{}); ok {
					extractSamplesAndNames(rpm, samples, names, locationInfos, mappingInfos, stackTable)
				}
			}
			return
		}
	}
	if scopes := getKey(m, "scopeProfiles", "ScopeProfiles", "scope_profiles"); scopes != nil {
		if arr, ok := scopes.([]interface{}); ok {
			for _, s := range arr {
				if sm, ok := s.(map[string]interface{}); ok {
					extractSamplesAndNames(sm, samples, names, locationInfos, mappingInfos, stackTable)
				}
			}
			return
		}
	}
	if profs := getKey(m, "profiles", "Profiles"); profs != nil {
		if arr, ok := profs.([]interface{}); ok {
			for _, p := range arr {
				if pm, ok := p.(map[string]interface{}); ok {
					extractSamplesAndNames(pm, samples, names, locationInfos, mappingInfos, stackTable)
				}
			}
			return
		}
	}
	// Process samples (names already filled from dictionary or scope).
	if sampleArr := getKey(m, "samples", "Samples", "sample", "Sample"); sampleArr != nil {
		locationIndices := getProfileLocationIndices(m)
		if arr, ok := sampleArr.([]interface{}); ok {
			for _, s := range arr {
				so, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				locIDs := getSampleLocIDs(so, locationIndices, stackTable)
				value := getSampleValue(so)
				if value <= 0 && len(locIDs) == 0 {
					continue
				}
				if value <= 0 {
					value = 1
				}
				stack := make([]string, 0, len(locIDs))
				for _, id := range locIDs {
					if name, ok := names[id]; ok && name != "" {
						stack = append(stack, name)
					} else {
						stack = append(stack, fmt.Sprintf("frame_%d", id))
					}
				}
				*samples = append(*samples, Sample{Stack: stack, Value: value, LocIndices: locIDs})
			}
		}
		return
	}
	for _, key := range []string{"resource", "scope", "profile", "Profile"} {
		if v := m[key]; v != nil {
			if vm, ok := v.(map[string]interface{}); ok {
				extractSamplesAndNames(vm, samples, names, locationInfos, mappingInfos, stackTable)
			}
		}
	}
}

// extractNamesFromDictionary fills names (location index -> symbol name) from OTLP 1.0 dictionary.
// Dictionary has stringTable, functionTable (nameStrindex -> stringTable), locationTable (line[].functionIndex -> functionTable).
func extractNamesFromDictionary(m map[string]interface{}, names map[int]string) {
	stringTable := getStringTable(m)
	if len(stringTable) == 0 {
		return
	}
	// functionTable: index -> name (from stringTable)
	funcNames := make(map[int]string)
	if ft := getKey(m, "functionTable", "FunctionTable", "function_table"); ft != nil {
		if arr, ok := ft.([]interface{}); ok {
			for idx, f := range arr {
				fm, _ := f.(map[string]interface{})
				if fm == nil {
					continue
				}
				if nameRef := getKey(fm, "nameStrindex", "nameStrIndex", "name_strindex", "name"); nameRef != nil {
					if s, ok := nameRef.(string); ok && s != "" {
						funcNames[idx] = s
					} else if i, ok := toInt(nameRef); ok && i >= 0 && i < len(stringTable) {
						funcNames[idx] = stringTable[i]
					}
				}
			}
		}
	}
	// locationTable: index -> name from first line's functionIndex
	if lt := getKey(m, "locationTable", "LocationTable", "location_table"); lt != nil {
		if arr, ok := lt.([]interface{}); ok {
			for idx, loc := range arr {
				lm, _ := loc.(map[string]interface{})
				if lm == nil {
					continue
				}
				if lineArr := getKey(lm, "line", "Line", "lines"); lineArr != nil {
					if lines, ok := lineArr.([]interface{}); ok && len(lines) > 0 {
						if first, ok := lines[0].(map[string]interface{}); ok {
							if fi := getKey(first, "functionIndex", "FunctionIndex", "function_index"); fi != nil {
								if fiIdx, ok := toInt(fi); ok && fiIdx >= 0 {
									if name := funcNames[fiIdx]; name != "" {
										names[idx] = name
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// extractLocationAndMappingTables fills locationInfos (location index -> mappingIndex, address) and
// mappingInfos (mapping index -> filename, build_id) from the OTLP dictionary for backend symbolization.
func extractLocationAndMappingTables(m map[string]interface{}, locationInfos map[int]LocationInfo, mappingInfos map[int]MappingInfo) {
	stringTable := getStringTable(m)
	// AttributeTable: index -> (key, value) via stringTable for build_id lookup on mappings.
	attrTable := getAttributeTable(m, stringTable)
	// LocationTable: index -> mappingIndex, address
	if lt := getKey(m, "locationTable", "LocationTable", "location_table"); lt != nil {
		if arr, ok := lt.([]interface{}); ok {
			for idx, loc := range arr {
				lm, _ := loc.(map[string]interface{})
				if lm == nil {
					continue
				}
				var info LocationInfo
				if mi := getKey(lm, "mappingIndex", "MappingIndex", "mapping_index"); mi != nil {
					if i, ok := toInt(mi); ok {
						info.MappingIndex = i
					}
				}
				if addr := getKey(lm, "address", "Address"); addr != nil {
					if u := toUint64(addr); u != nil {
						info.Address = *u
					}
				}
				locationInfos[idx] = info
			}
		}
	}
	// MappingTable: index -> filename (stringTable[filenameStrindex]), build_id (from attributeTable)
	if mt := getKey(m, "mappingTable", "MappingTable", "mapping_table"); mt != nil {
		if arr, ok := mt.([]interface{}); ok {
			for idx, mapping := range arr {
				mm, _ := mapping.(map[string]interface{})
				if mm == nil {
					continue
				}
				var info MappingInfo
				if fi := getKey(mm, "filenameStrindex", "FilenameStrindex", "filename_strindex"); fi != nil {
					if i, ok := toInt(fi); ok && i >= 0 && i < len(stringTable) {
						info.Filename = stringTable[i]
					}
				}
				if attrIndices := getKey(mm, "attributeIndices", "AttributeIndices", "attribute_indices"); attrIndices != nil {
					if indices, ok := attrIndices.([]interface{}); ok {
						for _, ai := range indices {
							if i, ok := toInt(ai); ok && i >= 0 {
								if kv, ok := attrTable[i]; ok && kv.Value != "" {
									if strings.Contains(strings.ToLower(kv.Key), "build_id") {
										info.BuildID = kv.Value
										break
									}
								}
							}
						}
					}
				}
				mappingInfos[idx] = info
			}
		}
	}
}

// attrEntry is a key-value from the dictionary's attribute table.
type attrEntry struct{ Key, Value string }

func getAttributeTable(m map[string]interface{}, stringTable []string) map[int]attrEntry {
	out := make(map[int]attrEntry)
	at := getKey(m, "attributeTable", "AttributeTable", "attribute_table")
	if at == nil {
		return out
	}
	arr, ok := at.([]interface{})
	if !ok {
		return out
	}
	for idx, v := range arr {
		vm, _ := v.(map[string]interface{})
		if vm == nil {
			continue
		}
		var key, val string
		if k := getKey(vm, "keyStrindex", "KeyStrindex", "key_strindex"); k != nil {
			if i, ok := toInt(k); ok && i >= 0 && i < len(stringTable) {
				key = stringTable[i]
			}
		}
		if vv := getKey(vm, "valueStrindex", "ValueStrindex", "value_strindex"); vv != nil {
			if i, ok := toInt(vv); ok && i >= 0 && i < len(stringTable) {
				val = stringTable[i]
			}
		}
		out[idx] = attrEntry{Key: key, Value: val}
	}
	return out
}

func toUint64(v interface{}) *uint64 {
	switch x := v.(type) {
	case float64:
		u := uint64(x)
		return &u
	case int:
		u := uint64(x)
		return &u
	case int64:
		u := uint64(x)
		return &u
	}
	return nil
}

func getStringTable(m map[string]interface{}) []string {
	st := getKey(m, "stringTable", "StringTable", "string_table")
	if st == nil {
		return nil
	}
	arr, ok := st.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		} else {
			out = append(out, "")
		}
	}
	return out
}

// extractStackTable builds stack_index -> location_indices (root-first) from OTLP v1development ProfilesDictionary.stack_table.
// Proto Stack has location_indices with first element = leaf; we reverse so caller gets root-first.
func extractStackTable(dict map[string]interface{}) stackTableMap {
	st := getKey(dict, "stackTable", "StackTable", "stack_table")
	if st == nil {
		return nil
	}
	arr, ok := st.([]interface{})
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make(stackTableMap)
	for idx, entry := range arr {
		em, _ := entry.(map[string]interface{})
		if em == nil {
			continue
		}
		li := getKey(em, "locationIndices", "LocationIndices", "location_indices")
		if li == nil {
			continue
		}
		locArr, ok := li.([]interface{})
		if !ok {
			continue
		}
		locs := make([]int, 0, len(locArr))
		for _, v := range locArr {
			if i, ok := toInt(v); ok {
				locs = append(locs, i)
			}
		}
		// Proto: first location is leaf. Reverse for root-first (flame graph convention).
		for i, j := 0, len(locs)-1; i < j; i, j = i+1, j-1 {
			locs[i], locs[j] = locs[j], locs[i]
		}
		out[idx] = locs
	}
	return out
}

// getProfileLocationIndices returns the profile's locationIndices slice (indices into dictionary locationTable).
func getProfileLocationIndices(m map[string]interface{}) []int {
	li := getKey(m, "locationIndices", "LocationIndices", "location_indices")
	if li == nil {
		return nil
	}
	arr, ok := li.([]interface{})
	if !ok {
		return nil
	}
	out := make([]int, 0, len(arr))
	for _, v := range arr {
		if i, ok := toInt(v); ok {
			out = append(out, i)
		}
	}
	return out
}

// extractNamesFromObject fills names from stringTable, location, function in this object.
// Call this before recursing into "profiles" so scope-level dictionary is used when parsing profile samples.
func extractNamesFromObject(m map[string]interface{}, names map[int]string) {
	if st := getKey(m, "stringTable", "StringTable", "string_table"); st != nil {
		if arr, ok := st.([]interface{}); ok {
			for i, v := range arr {
				if s, ok := v.(string); ok {
					names[i] = s
				}
			}
		}
	}
	if locs := getKey(m, "location", "Location", "locations"); locs != nil {
		if arr, ok := locs.([]interface{}); ok {
			for idx, loc := range arr {
				name := resolveLocationName(loc, names)
				if name != "" {
					names[idx] = name
				} else {
					names[idx] = fmt.Sprintf("loc_%d", idx)
				}
			}
		}
	}
	if fncs := getKey(m, "function", "Function", "functions"); fncs != nil {
		if arr, ok := fncs.([]interface{}); ok {
			for idx, fn := range arr {
				name := resolveFunctionName(fn, names)
				if name != "" {
					names[idx] = name
				}
			}
		}
	}
}

func resolveLocationName(loc interface{}, names map[int]string) string {
	lm, ok := loc.(map[string]interface{})
	if !ok {
		return ""
	}
	if nameRef := getKey(lm, "name", "Name", "functionName", "function_name"); nameRef != nil {
		if s, ok := nameRef.(string); ok && s != "" {
			return s
		}
		if idx, ok := toInt(nameRef); ok && idx >= 0 {
			return names[idx]
		}
	}
	if lineArr := getKey(lm, "line", "Line", "lines"); lineArr != nil {
		if arr, ok := lineArr.([]interface{}); ok && len(arr) > 0 {
			first := arr[0]
			if fm, ok := first.(map[string]interface{}); ok {
				if funcIdx := getKey(fm, "functionIndex", "FunctionIndex", "function_index"); funcIdx != nil {
					if idx, ok := toInt(funcIdx); ok && idx >= 0 {
						return names[idx]
					}
				}
			}
		}
	}
	return ""
}

func resolveFunctionName(fn interface{}, names map[int]string) string {
	fm, ok := fn.(map[string]interface{})
	if !ok {
		return ""
	}
	if nameRef := getKey(fm, "name", "Name"); nameRef != nil {
		if s, ok := nameRef.(string); ok && s != "" {
			return s
		}
		if idx, ok := toInt(nameRef); ok && idx >= 0 {
			return names[idx]
		}
	}
	return ""
}

func toInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	}
	return 0, false
}

// getSampleLocIDs returns location table indices for the sample (root-first order).
// locationIndices is the profile's locationIndices (OTEP format); pass nil if not used.
// stackTable is v1development: stack_index -> location_indices (root-first); pass nil if not used.
func getSampleLocIDs(so map[string]interface{}, locationIndices []int, stackTable stackTableMap) []int {
	// v1development: Sample.stack_index references ProfilesDictionary.stack_table; resolve to location indices (root-first).
	if stackIdx := getKey(so, "stackIndex", "stack_index"); stackIdx != nil {
		if idx, ok := toInt(stackIdx); ok && stackTable != nil {
			if locs := stackTable[idx]; len(locs) > 0 {
				return locs
			}
		}
	}
	// OTEP: locationsStartIndex + locationsLength (camelCase / snake_case for collector JSON). Return root-first.
	if start := getKey(so, "locationsStartIndex", "LocationsStartIndex", "locations_start_index"); start != nil {
		if startIdx, ok := toInt(start); ok && startIdx >= 0 {
			length := 1
			if l := getKey(so, "locationsLength", "LocationsLength", "locations_length"); l != nil {
				if n, ok := toInt(l); ok && n > 0 {
					length = n
				}
			}
			ids := make([]int, 0, length)
			for i := startIdx; i < startIdx+length; i++ {
				locIdx := i
				if i < len(locationIndices) {
					locIdx = locationIndices[i]
				}
				ids = append(ids, locIdx)
			}
			// Reverse to root-first (leaf was at end in OTLP)
			for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
				ids[i], ids[j] = ids[j], ids[i]
			}
			return ids
		}
	}
	if locArray := getKey(so, "attributeIndices", "attribute_indices", "locationIdList", "location_id_list"); locArray != nil {
		if arr, ok := locArray.([]interface{}); ok {
			ids := make([]int, 0, len(arr))
			for i := len(arr) - 1; i >= 0; i-- {
				if idx, ok := toInt(arr[i]); ok {
					ids = append(ids, idx)
				}
			}
			return ids
		}
	}
	if locID := getKey(so, "locationId", "LocationId", "location_id"); locID != nil {
		if idx, ok := toInt(locID); ok {
			return []int{idx}
		}
	}
	return nil
}

func getSampleValue(so map[string]interface{}) int64 {
	if v := getKey(so, "value", "Value", "values"); v != nil {
		if n, ok := v.(float64); ok {
			return int64(n)
		}
		if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			var sum int64
			for _, a := range arr {
				sum += toInt64(a)
			}
			return sum
		}
	}
	if ts := getKey(so, "timestampsUnixNano", "timestamps_unix_nano"); ts != nil {
		if arr, ok := ts.([]interface{}); ok {
			return int64(len(arr))
		}
	}
	return 1
}
