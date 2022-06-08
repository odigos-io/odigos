package inspectors

import (
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	"github.com/keyval-dev/odigos/langDetector/process"
)

type inspector interface {
	Inspect(process *process.Details) (v1.ProgrammingLanguage, bool)
}

var inspectorsList = []inspector{java, python, dotNet, nodeJs, golang}

func DetectLanguage(processes []*process.Details) []v1.ProgrammingLanguage {
	var result []v1.ProgrammingLanguage
	for _, p := range processes {
		for _, i := range inspectorsList {
			inspectionResult, detected := i.Inspect(p)
			if detected {
				result = append(result, inspectionResult)
				break
			}
		}
	}

	return result
}
