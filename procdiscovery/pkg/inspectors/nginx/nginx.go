package nginx

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

func (j *NginxInspector) LightCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if (filepath.Base(ctx.ExePath) == NginxProcessName) || strings.Contains(ctx.CmdLine, NginxProcessName) {
		return common.NginxProgrammingLanguage, true
	}

	return "", false
}

func (j *NginxInspector) ExpensiveCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (j *NginxInspector) GetRuntimeVersion(ctx *process.ProcessContext, containerURL string) *version.Version {
	nginxVersion, err := GetNginxVersion(containerURL)
	if err != nil {
		return nil
	}

	return common.GetVersion(nginxVersion)
}

func GetNginxVersion(containerURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, containerURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
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
