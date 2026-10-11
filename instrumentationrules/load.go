package instrumentationrules

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/*
var instrumentationRulesFS embed.FS

// array of all instrumentation-rule catalog configs
var loadedRules []InstrumentationRule

// map from rule type to rule catalog config object
var rulesByType map[string]InstrumentationRule

func Load() error {
	return load(instrumentationRulesFS)
}

func Get() []InstrumentationRule {
	return loadedRules
}

func GetRuleByType(ruleType string) (InstrumentationRule, bool) {
	rule, ok := rulesByType[ruleType]
	return rule, ok
}

func load(fs embed.FS) error {
	var rules []InstrumentationRule
	var rulesByTypeMap = make(map[string]InstrumentationRule)

	files, err := fs.ReadDir("data")
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := file.Name()

		if !strings.HasSuffix(fileName, ".yaml") && !strings.HasSuffix(fileName, ".yml") {
			continue
		}

		bytesData, err := fs.ReadFile("data/" + file.Name())
		if err != nil {
			return err
		}

		var rule InstrumentationRule
		err = yaml.Unmarshal(bytesData, &rule)
		if err != nil {
			return err
		}

		rulesByTypeMap[rule.Metadata.Type] = rule
		rules = append(rules, rule)
	}

	rulesByType = rulesByTypeMap
	loadedRules = rules
	return nil
}
