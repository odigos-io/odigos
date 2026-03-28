package collectorprofiles

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

var attrLineRe = regexp.MustCompile(`^\s+->\s+(.+?):\s+Str\((.*)\)`)

type GwDumpParsed struct {
	StringTable []string
	Resources   []GwDumpResource
	StackLines  []string
}

type GwDumpResource struct {
	Attributes  map[string]string
	SampleCount int
}

func ParseGwProfilesDump(path string) (*GwDumpParsed, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := &GwDumpParsed{StringTable: []string{""}}
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var inStringTable bool
	var inResourceBlock bool
	var currentResource *GwDumpResource
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
			} else if strings.HasPrefix(line, "    ") && trimmed != "" {
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
				if m := attrLineRe.FindStringSubmatch(line); len(m) >= 3 {
					currentResource.Attributes[strings.TrimSpace(m[1])] = m[2]
				}
				continue
			}
			if strings.HasPrefix(trimmed, "Sample #") {
				currentResource.SampleCount++
				continue
			}
			if strings.HasPrefix(line, "ResourceProfiles #") || (trimmed != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "ScopeProfiles") && !strings.HasPrefix(line, "Profile #") && !strings.HasPrefix(line, "InstrumentationScope") && trimmed != "Resource attributes:") {
				if currentResource != nil && len(currentResource.Attributes) > 0 {
					out.Resources = append(out.Resources, *currentResource)
					currentResource = nil
				}
				inResourceBlock = false
			}
			continue
		}

		if !inStringTable && len(out.Resources) == 0 && strings.HasPrefix(line, "    ") && trimmed != "" {
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

func (p *GwDumpParsed) ToOTLPJSON(useFirstResource bool, maxSamples int) ([]byte, error) {
	if len(p.StringTable) == 0 {
		p.StringTable = []string{""}
	}
	nameIdx := 1
	if len(p.StringTable) > 1 {
		for i := 1; i < len(p.StringTable); i++ {
			if p.StringTable[i] != "" && !strings.HasPrefix(p.StringTable[i], "thread.") {
				nameIdx = i
				break
			}
		}
	}
	stackTable := []map[string]interface{}{{"locationIndices": []int{0}}}
	locationTable := []map[string]interface{}{{"mappingIndex": 0, "address": 0, "lines": []map[string]interface{}{{"functionIndex": 0}}}}
	functionTable := []map[string]interface{}{{"nameStrindex": nameIdx, "systemNameStrindex": 0, "filenameStrindex": 0, "startLine": 0}}

	samples := []map[string]interface{}{}
	n := maxSamples
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		samples = append(samples, map[string]interface{}{
			"stackIndex":         0,
			"values":             []int64{1},
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
						"scope": map[string]interface{}{"name": "go.opentelemetry.io/ebpf-profiler"},
						"profiles": []map[string]interface{}{
							{
								"sampleType":   map[string]interface{}{"typeStrindex": 0, "unitStrindex": 1},
								"timeUnixNano": "1000000000000000000",
								"periodType":   map[string]interface{}{"typeStrindex": 0, "unitStrindex": 1},
								"period":       "1",
								"samples":      samples,
							},
						},
					},
				},
			},
		},
		"dictionary": map[string]interface{}{
			"stringTable":   p.StringTable,
			"stackTable":    stackTable,
			"locationTable": locationTable,
			"functionTable": functionTable,
			"mappingTable":  []map[string]interface{}{{"memoryStart": 0, "memoryLimit": 0, "fileOffset": 0, "filenameStrindex": 0}},
		},
	}
	return json.Marshal(root)
}
