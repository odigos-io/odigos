package inspectors

import (
	"fmt"
	"github.com/keyval-dev/odigos/langDetector/inspectors/goversion"
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	"github.com/keyval-dev/odigos/langDetector/process"
	"io/fs"
	"os"
	"runtime"
	"strings"
)

type golangInspector struct{}

var golang = &golangInspector{}

func (g *golangInspector) Inspect(p *process.Details) (v1.ProgrammingLanguage, bool) {
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	info, err := os.Stat(file)
	if err != nil {
		fmt.Printf("could not perform os.stat: %s\n", err)
		return "", false
	}

	if !isExe(file, info) {
		fmt.Printf("isExe returned false\n")
		return "", false
	}

	x, err := goversion.OpenExe(file)
	if err != nil {
		fmt.Printf("could not perform OpenExe: %s\n", err)
		return "", false
	}

	if x.Elf().Section(".gosymtab") == nil {
		return "", false
	}

	vers, _ := goversion.FindVersion(x)
	if vers == "" {
		// Not a golang app
		return "", false
	}

	return v1.GoProgrammingLanguage, true
}

// isExe reports whether the file should be considered executable.
func isExe(file string, info fs.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(file), ".exe")
	}
	return info.Mode().IsRegular() && info.Mode()&0111 != 0
}
