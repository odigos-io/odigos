//go:build odigletminimal
// +build odigletminimal

package inspectors

import "github.com/odigos-io/odigos/common"

var disabledLanguages = map[common.ProgrammingLanguage]struct{}{
	common.DotNetProgrammingLanguage: {},
	common.PhpProgrammingLanguage:    {},
	common.RubyProgrammingLanguage:   {},
}
