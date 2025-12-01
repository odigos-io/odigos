package cplusplus

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type CPlusPlusInspector struct{}

var (
	binaries = []string{
		"libstdc++.so", // Unique to C++ (specifically, GCC)
		"libc++.so",    // Unique to C++ (specifically, Clang/LLVM)
	}
)

func (n *CPlusPlusInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *CPlusPlusInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	mapsFile, err := pcx.GetMapsFile()
	if err != nil {
		return "", false
	}

	if utils.IsMapsFileContainsBinary(mapsFile, binaries) {
		return common.CPlusPlusProgrammingLanguage, true
	}

	return "", false
}

func (n *CPlusPlusInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	return ""
}
