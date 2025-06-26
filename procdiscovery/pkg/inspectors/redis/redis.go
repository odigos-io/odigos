package redis

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RedisInspector struct{}

var (
	processNames = []string{"redis", "redis-server", "redis-sentinel", "redis-cli"}
)

func (n *RedisInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if utils.IsProcessEqualProcessNames(pcx, processNames) {
		return common.RedisProgrammingLanguage, true
	}

	return "", false
}

func (n *RedisInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
