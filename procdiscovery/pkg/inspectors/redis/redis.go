package redis

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RedisInspector struct{}

var (
	processNames = []string{"redis", "redis-server", "redis-sentinel", "redis-cli"}
)

func (n *RedisInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	if utils.IsBaseExeContainsProcessName(baseExe, processNames) {
		return common.RedisProgrammingLanguage, true
	}

	return "", false
}

func (n *RedisInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
