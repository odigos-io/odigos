package envOverwrite

import (
	"github.com/keyval-dev/odigos/common"
)

type Overwriter interface {
	EnvName() string
	ValueFor(sdkType common.OtelSdkType) string
	Revert(str string) string
}

var all = []Overwriter{
	&nodeOptions{},
}

var byName = map[string]Overwriter{}

func loadToMap() {
	for _, o := range all {
		byName[o.EnvName()] = o
	}
}

func ShouldOverwrite(envName string) bool {
	if len(byName) == 0 {
		loadToMap()
	}

	_, ok := byName[envName]
	return ok
}

func Patch(envName string, currentVal string, sdkType common.OtelSdkType) string {
	if len(byName) == 0 {
		loadToMap()
	}

	o, exists := byName[envName]
	if !exists {
		return ""
	}

	additionalVal := o.ValueFor(sdkType)
	if currentVal == "" {
		return additionalVal
	}

	return currentVal + " " + additionalVal
}

func Revert(envName string, currentVal string) string {
	if len(byName) == 0 {
		loadToMap()
	}

	o, exists := byName[envName]
	if !exists {
		return ""
	}

	return o.Revert(currentVal)
}
