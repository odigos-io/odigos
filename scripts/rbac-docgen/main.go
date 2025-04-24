package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
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
title: "Kubernetes RBAC Permissions"
sidebarTitle: "Kubernetes Permissions"
---

This page lists the Kubernetes Roles and ClusterRoles used by Odigos and the Odigos Operator.

`

	cmd := exec.Command("helm", "template", "odigos", "../../helm/odigos", "--set", "openshift.enabled=true")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	result := string(out)
	manifests := strings.Split(result, "---")
	scheme := runtime.NewScheme()
	_ = rbacv1.AddToScheme(scheme)

	decoder := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)
	roles := make([]*rbacv1.Role, 0)
	clusterRoles := make([]*rbacv1.ClusterRole, 0)
	for _, manifest := range manifests {
		obj, _, err := decoder.Decode([]byte(manifest), nil, nil)
		if err != nil {
			continue // ignore unknown types
		}

		switch v := obj.(type) {
		case *rbacv1.Role:
			roles = append(roles, v)
		case *rbacv1.ClusterRole:
			clusterRoles = append(clusterRoles, v)
		}
	}

	docString += "# Components\n\nThis section lists the RBAC policies used by the Odigos components.\n\n"
	docString += "## ClusterRoles\n\nBelow are the ClusterRoles used by Odigos components.\n\n"

	for _, cr := range clusterRoles {
		docString += "### " + cr.GetName() + "\n\n"
		docString += parseRbacRules(cr.Rules)
	}

	docString += "## Roles\n\nBelow are the Roles used by Odigos components.\n\n"

	for _, r := range roles {
		docString += "### " + r.GetName() + "\n\n"
		docString += parseRbacRules(r.Rules)
	}

	docString += "# Operator\n\nThis section lists the RBAC policies used by the Odigos Operator. Many of these permissions are necessary in order to create the RBAC policies for the components listed above.\n\n"
	docString += "## ClusterRoles\n\n"

	for _, permission := range csv.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
		docString += parseRbacRules(permission.Rules)
	}

	err = os.WriteFile("../../docs/permissions.mdx", []byte(docString), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println(docString)
}

func parseRbacRules(rules []rbacv1.PolicyRule) string {
	docString := "| APIGroups | Resources | Resource Names | Verbs |\n"
	docString += "|---|---|---|---|"
	for _, rule := range rules {
		docString = docString + "\n|"
		docString += parseRuleField(rule.APIGroups)
		docString += parseRuleField(rule.Resources)
		docString += parseRuleField(rule.ResourceNames)
		docString += parseRuleField(rule.Verbs)
	}
	docString += "\n\n"
	return docString
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
			docString += "<br />"
		}
	}
	docString += " |"
	return docString
}
