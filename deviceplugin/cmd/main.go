package main

import (
	"github.com/odigos-io/odigos/deviceplugin/pkg"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
)

func main() {
	if err := log.Init(); err != nil {
		panic(err)
	}

	dp := pkg.New()

	if err := dp.Run(); err != nil {
		log.Logger.Error(err, "Device plugin exited with error")
	}
}
