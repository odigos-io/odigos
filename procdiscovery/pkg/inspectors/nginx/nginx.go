package nginx

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type NginxInspector struct{}

const (
	NginxProcessName  = "nginx"
	NginxVersionRegex = `nginx/(\d+\.\d+\.\d+)`
)

var re = regexp.MustCompile(NginxVersionRegex)

func (j *NginxInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(p.CmdLine, NginxProcessName) || strings.Contains(p.ExePath, NginxProcessName) {
		return common.NginxProgrammingLanguage, true
	}

	return "", false
}

func (j *NginxInspector) GetRuntimeVersion(p *process.Details, containerURL string) *version.Version {
	nginxVersion, err := GetNginxVersion(containerURL)
	if err != nil {
		return nil
	}

	return common.GetVersion(nginxVersion)
}

func GetNginxVersion(containerURL string) (string, error) {
	resp, err := http.Get(containerURL)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	serverHeader := resp.Header.Get("Server")
	if serverHeader == "" {
		return "", nil
	}

	match := re.FindStringSubmatch(serverHeader)
	if len(match) != 2 {
		return "", nil
	}

	return match[1], nil
}
