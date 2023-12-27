package inspectors

import (
	"fmt"
	"os"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/inspectors/goversion"
	procdiscovery "github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type golangInspector struct{}

var golang = &golangInspector{}

func (g *golangInspector) Inspect(p *procdiscovery.Details) (common.ProgrammingLanguage, bool) {
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	_, err := os.Stat(file)
	if err != nil {
		fmt.Printf("could not perform os.stat: %s\n", err)
		return "", false
	}

	x, err := goversion.OpenExe(file)
	if err != nil {
		fmt.Printf("could not perform OpenExe: %s\n", err)
		return "", false
	}

	vers, _ := goversion.FindVersion(x)
	if vers == "" {
		// Not a golang app
		return "", false
	}

	return common.GoProgrammingLanguage, true
}
