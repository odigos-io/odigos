package services

import "time"

type versionedModules struct {
	Timestamp time.Time     `json:"timestamp"`
	Mods      []*jsonModule `json:"mods"`
}

type jsonModule struct {
	Module   string         `json:"module"`
	Packages []*jsonPackage `json:"packages"`
}

type jsonPackage struct {
	Package string        `json:"package"`
	Structs []*jsonStruct `json:"structs"`
}

type jsonStruct struct {
	Struct    string       `json:"struct"`
	Anonymous bool         `json:"anonymous,omitempty"`
	Fields    []*jsonField `json:"fields"`
}

type jsonField struct {
	Field   string        `json:"field"`
	Offsets []*jsonOffset `json:"offsets"`
}

type jsonOffset struct {
	Offset   *uint64  `json:"offset"`
	Versions []string `json:"versions"`
}
