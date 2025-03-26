package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	semver "github.com/Masterminds/semver/v3"
)

var (
	//go:embed offset_results.json
	offsetsData string
)

type Module struct {
	ModuleName string       `json:"module"`
	Packages   []ModPackage `json:"packages"`
}

type ModPackage struct {
	PackageName string           `json:"package"`
	Structs     []PackageStructs `json:"structs"`
}

type PackageStructs struct {
	StructName string  `json:"struct"`
	Fields     []Field `json:"fields"`
}

type Field struct {
	FieldName string        `json:"field"`
	Offsets   []FieldOffset `json:"offsets"`
}

type FieldOffset struct {
	Offset   *int     `json:"offset"`
	Versions []string `json:"versions"`
}

func main() {
	var data []Module

	err := json.Unmarshal([]byte(offsetsData), &data)
	if err != nil {
		panic(err)
	}

	moduleVersions := make(map[string]map[string]struct{})

	for _, module := range data {
		moduleMap := make(map[string]struct{})
		for _, p := range module.Packages {
			for _, s := range p.Structs {
				for _, f := range s.Fields {
					for _, o := range f.Offsets {
						for _, v := range o.Versions {
							if _, ok := moduleMap[v]; !ok {
								moduleMap[v] = struct{}{}
							}
						}
					}
				}
			}
		}
		moduleVersions[module.ModuleName] = moduleMap
	}

	docsHeader := `---
title: "Go Library Supported Versions"
sidebarTitle: "Supported Library Versions"
---

This page shows the versions of Go and all instrumented libraries that Odigos supports.

`

	docsOutput := docsHeader

	// Print Go versions at the top of the doc
	goVersionOutput, err := outputVersions("std", moduleVersions["std"])
	if err != nil {
		panic(err)
	}
	docsOutput = docsOutput + goVersionOutput + "\n"

	// sort sortedModules so they are always printed in the same order
	sortedModules := make([]string, 0)
	for module, _ := range moduleVersions {
		sortedModules = append(sortedModules, module)
	}
	slices.Sort(sortedModules)

	for _, module := range sortedModules {
		if module == "std" {
			continue
		}
		output, err := outputVersions(module, moduleVersions[module])
		if err != nil {
			panic(err)
		}
		docsOutput = docsOutput + output + "\n"
	}

	fmt.Println(docsOutput)
}

func outputVersions(name string, versionsMap map[string]struct{}) (string, error) {
	if name == "std" {
		name = "Go versions (standard libraries)"
	} else {
		name = "`" + name + "`"
	}
	versionsOutput := "## " + name + "\n\n"

	// sort versions
	sortedVersions := make([]*semver.Version, 0)
	for v, _ := range versionsMap {
		version, err := semver.NewVersion(v)
		if err != nil {
			return "", err
		}
		sortedVersions = append(sortedVersions, version)
	}
	sort.Sort(semver.Collection(sortedVersions))

	for _, v := range sortedVersions {
		versionsOutput = versionsOutput + "* " + v.String() + "\n"
	}

	return versionsOutput, nil
}
