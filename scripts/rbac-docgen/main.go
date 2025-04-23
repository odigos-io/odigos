package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

func main() {
	// Load the YAML file
	data, err := os.ReadFile("../../operator/bundle/manifests/odigos-operator.clusterserviceversion.yaml")
	if err != nil {
		panic(err)
	}

	var csv v1alpha1.ClusterServiceVersion
	err = yaml.Unmarshal(data, &csv)
	if err != nil {
		panic(err)
	}

	docString := `---
title: "Operator Permissions"
sidebarTitle: "Operator Permissions"
---

This page lists the cluster roles used by the Odigos Operator.

`
	docString += "| APIGroups | Resources | Resource Names | Verbs |\n"
	docString += "|---|---|---|---|"

	for _, permission := range csv.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
		for _, rule := range permission.Rules {
			docString = docString + "\n|"
			docString += parseRuleField(rule.APIGroups)
			docString += parseRuleField(rule.Resources)
			docString += parseRuleField(rule.ResourceNames)
			docString += parseRuleField(rule.Verbs)
		}
	}

	err = os.WriteFile("../../docs/permissions.mdx", []byte(docString), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println(docString)
}

func parseRuleField(list []string) string {
	docString := " "
	if len(list) == 0 {
		return " \\* |"
	}
	for i, value := range list {
		if len(value) == 0 || value == "*" {
			value = "\\*"
		}
		docString += value
		if i < len(list)-1 {
			docString += ","
		}
	}
	docString += " |"
	return docString
}
