package collectorprofiles

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// GwDumpParsed holds data extracted from gw-profiles-dump.logs (gateway debug exporter text log).
type GwDumpParsed struct {
	StringTable []string              // From "String table:" section (index 0 = "")
	Resources   []GwDumpResource      // From each "ResourceProfiles #N" block
	StackLines  []string              // Resolved stack frame lines (between string table and ResourceProfiles)
}

// GwDumpResource is one ResourceProfiles block: attributes and sample count.
type GwDumpResource struct {
	Attributes map[string]string // e.g. "k8s.namespace.name" -> "odigos-system"
	SampleCount int              // Number of "Sample #N" in this block
}

// ParseGwProfilesDump reads gw-profiles-dump.logs and extracts string table, resources, and stack lines.
// The log format: "String table:" then lines "    <string>", then resolved stack frames (one per line),
// then "ResourceProfiles #N", "Resource attributes:", "     -> key: Str(value)", "Sample #M", etc.
func ParseGwProfilesDump(path string) (*GwDumpParsed, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := &GwDumpParsed{
		StringTable: []string{""},
		Resources:   nil,
		StackLines:  nil,
	}
	scanner := bufio.NewScanner(f)
	// We don't know max line length; use a large buffer for long lines.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var inStringTable bool
	var inResourceBlock bool
	var currentResource *GwDumpResource
	// Match "     -> k8s.namespace.name: Str(odigos-system)" or "     -> key: Str()"
	attrRe := regexp.MustCompile(`^\s+->\s+(.+?):\s+Str\((.*)\)`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(line, "String table:") {
			inStringTable = true
			inResourceBlock = false
			continue
		}
		if inStringTable {
			if strings.HasPrefix(line, "ResourceProfiles #") {
				inStringTable = false
				// Fall through to handle ResourceProfiles
			} else if strings.HasPrefix(line, "    ") && trimmed != "" {
				// String table entry (4-space indent)
				out.StringTable = append(out.StringTable, strings.TrimSpace(line))
				continue
			} else if trimmed != "" && !strings.HasPrefix(line, "    ") {
				inStringTable = false
			} else {
				continue
			}
		}

		if strings.HasPrefix(line, "ResourceProfiles #") {
			if currentResource != nil {
				out.Resources = append(out.Resources, *currentResource)
			}
			currentResource = &GwDumpResource{Attributes: make(map[string]string)}
			inResourceBlock = true
			continue
		}
		if inResourceBlock {
			if strings.HasPrefix(line, "     -> ") {
				if m := attrRe.FindStringSubmatch(line); len(m) >= 3 {
					currentResource.Attributes[strings.TrimSpace(m[1])] = m[2]
				}
				continue
			}
			if strings.HasPrefix(trimmed, "Sample #") {
				currentResource.SampleCount++
				continue
			}
			if strings.HasPrefix(line, "ResourceProfiles #") || (trimmed != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "ScopeProfiles") && !strings.HasPrefix(line, "Profile #") && !strings.HasPrefix(line, "InstrumentationScope") && trimmed != "Resource attributes:") {
				// End of this resource block (e.g. next section)
				if currentResource != nil && len(currentResource.Attributes) > 0 {
					out.Resources = append(out.Resources, *currentResource)
					currentResource = nil
				}
				inResourceBlock = false
			}
			continue
		}

		// Between string table and first ResourceProfiles: resolved stack lines (4-space indent, look like symbols)
		if inStringTable == false && len(out.Resources) == 0 && strings.HasPrefix(line, "    ") && trimmed != "" {
			if !strings.HasPrefix(trimmed, "->") && !strings.HasPrefix(trimmed, "Resource") {
				out.StackLines = append(out.StackLines, strings.TrimSpace(line))
			}
		}
	}
	if currentResource != nil && len(currentResource.Attributes) > 0 {
		out.Resources = append(out.Resources, *currentResource)
	}
	return out, scanner.Err()
}

// ToOTLPJSON builds a minimal OTLP profile JSON chunk from parsed gw dump so we can run it through
// BuildPyroscopeProfileFromChunks. Uses string table from the log; adds minimal dictionary (stack_table,
// location_table, function_table) so one sample with stackIndex 0 resolves; uses first resource's
// attributes and creates one sample per StackLine (or a single synthetic sample if no stack lines).
func (p *GwDumpParsed) ToOTLPJSON(useFirstResource bool, maxSamples int) ([]byte, error) {
	if len(p.StringTable) == 0 {
		p.StringTable = []string{""}
	}
	// Build minimal dictionary: stringTable, one function (index 1 = first symbol), one location, one stack.
	// So stack_table[0] = location_indices [0], location_table[0] = line with function 0, function_table[0] = name 1.
	nameIdx := 1
	if len(p.StringTable) > 1 {
		// Use first non-empty string as sample type / symbol
		for i := 1; i < len(p.StringTable); i++ {
			if p.StringTable[i] != "" && !strings.HasPrefix(p.StringTable[i], "thread.") {
				nameIdx = i
				break
			}
		}
	}
	// stack_table: one stack with one location (location index 0)
	// location_table: one location with one line, function_index 0
	// function_table: one function with name_strindex = nameIdx
	stackTable := []map[string]interface{}{
		{"locationIndices": []int{0}},
	}
	locationTable := []map[string]interface{}{
		{"mappingIndex": 0, "address": 0, "lines": []map[string]interface{}{{"functionIndex": 0}}},
	}
	functionTable := []map[string]interface{}{
		{"nameStrindex": nameIdx, "systemNameStrindex": 0, "filenameStrindex": 0, "startLine": 0},
	}
	samples := []map[string]interface{}{}
	n := maxSamples
	if n <= 0 {
		n = 1
	}
	// Use stack lines as samples: each stack line becomes one sample with value 1, all using stack 0 (single frame)
	for i := 0; i < n; i++ {
		samples = append(samples, map[string]interface{}{
			"stackIndex":        0,
			"values":            []int64{1},
			"timestampsUnixNano": []string{"1000000000000000000"},
		})
	}
	resource := map[string]interface{}{
		"attributes": []map[string]interface{}{
			{"key": "k8s.namespace.name", "value": map[string]interface{}{"stringValue": "odigos-system"}},
			{"key": "k8s.daemonset.name", "value": map[string]interface{}{"stringValue": "odiglet"}},
			{"key": "service.name", "value": map[string]interface{}{"stringValue": "odiglet"}},
		},
	}
	if useFirstResource && len(p.Resources) > 0 {
		attrs := []map[string]interface{}{}
		for k, v := range p.Resources[0].Attributes {
			attrs = append(attrs, map[string]interface{}{"key": k, "value": map[string]interface{}{"stringValue": v}})
		}
		if len(attrs) > 0 {
			resource["attributes"] = attrs
		}
	}
	root := map[string]interface{}{
		"resourceProfiles": []map[string]interface{}{
			{
				"resource": resource,
				"scopeProfiles": []map[string]interface{}{
					{
						"scope":   map[string]interface{}{"name": "go.opentelemetry.io/ebpf-profiler"},
						"profiles": []map[string]interface{}{
							{
								"sampleType":    map[string]interface{}{"typeStrindex": 0, "unitStrindex": 1},
								"timeUnixNano":  "1000000000000000000",
								"periodType":    map[string]interface{}{"typeStrindex": 0, "unitStrindex": 1},
								"period":        "1",
								"samples":      samples,
							},
						},
					},
				},
			},
		},
		"dictionary": map[string]interface{}{
			"stringTable":    p.StringTable,
			"stackTable":     stackTable,
			"locationTable":  locationTable,
			"functionTable":  functionTable,
			"mappingTable":   []map[string]interface{}{{"memoryStart": 0, "memoryLimit": 0, "fileOffset": 0, "filenameStrindex": 0}},
		},
	}
	return json.Marshal(root)
}
