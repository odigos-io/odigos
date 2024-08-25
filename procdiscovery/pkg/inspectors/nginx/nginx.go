package nginx

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type NginxInspector struct{}

const NginxProcessName = "nginx"

func (j *NginxInspector) Inspect(p *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails
	if strings.Contains(p.CmdLine, NginxProcessName) || strings.Contains(p.ExeName, NginxProcessName) {
		programLanguageDetails.Language = common.NginxProgrammingLanguage
		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}

func GetNginxVersion(podIP string) (string, error) {
	resp, err := http.Get("http://" + podIP)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	serverHeader := resp.Header.Get("Server")
	if serverHeader == "" {
		return "", nil
	}

	re := regexp.MustCompile(`nginx/(\d+\.\d+\.\d+)`)
	match := re.FindStringSubmatch(serverHeader)
	if len(match) != 2 {
		return "", nil
	}

	return match[1], nil
}
