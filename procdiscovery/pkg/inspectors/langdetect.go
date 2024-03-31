package inspectors

import (
	"errors"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/dotnet"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/golang"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/java"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/mysql"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/nodejs"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/python"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type inspector interface {
	Inspect(process *process.Details) (common.ProgrammingLanguage, bool)
}

var inspectorsList = []inspector{
	&golang.GolangInspector{},
	&java.JavaInspector{},
	&python.PythonInspector{},
	&dotnet.DotnetInspector{},
	&nodejs.NodejsInspector{},
	&mysql.MySQLInspector{},
}

var ErrLanguageNotDetected = errors.New("language not detected")

// DetectLanguage returns the detected language for the process
func DetectLanguage(process process.Details) (common.ProgrammingLanguage, error) {
	for _, i := range inspectorsList {
		language, detected := i.Inspect(&process)
		if detected {
			return language, nil
		}
	}

	return common.UnknownProgrammingLanguage, ErrLanguageNotDetected
}
