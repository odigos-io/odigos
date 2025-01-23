package utils

import (
	"fmt"
	"runtime"
)

const (
	PackageID      = "lm-data-sdk-go"
	PackageVersion = "1.3.0"
)

func BuildUserAgent() string {
	return fmt.Sprintf("%s/%s;%s;%s;%s",
		PackageID, PackageVersion,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
