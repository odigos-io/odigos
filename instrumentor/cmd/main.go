/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	"github.com/odigos-io/odigos/instrumentor/sdks"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "net/http/pprof"
)

func main() {
	commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL"))
	logger := commonlogger.Logger()

	managerOptions := controllers.KubeManagerOptions{}
	var telemetryDisabled bool

	flag.StringVar(&managerOptions.MetricsServerBindAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&managerOptions.HealthProbeBindAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&managerOptions.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&telemetryDisabled, "telemetry-disabled", false, "Disable telemetry")
	flag.Parse()

	ctrl.SetLogger(commonlogger.FromSlogHandler())
	managerOptions.Logger = commonlogger.FromSlogHandler()

	// TODO: remove once the webhook stops using the default SDKs from the sdks package
	sdks.SetDefaultSDKs()

	logger.Info("Starting Odigos Community Instrumentor")

	distrosGetter, err := distros.NewCommunityGetter()
	if err != nil {
		logger.Error("Failed to initialize distro getter", "err", err)
		os.Exit(1)
	}
	dp, err := distros.NewProvider(distros.NewCommunityDefaulter(), distrosGetter)
	if err != nil {
		logger.Error("Failed to initialize distro provider", "err", err)
		os.Exit(1)
	}

	i, err := instrumentor.New(managerOptions, dp, nil)
	if err != nil {
		logger.Error("Failed to initialize instrumentor", "err", err)
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()

	i.Run(ctx, telemetryDisabled)
	logger.Info("instrumentor exiting")
}
