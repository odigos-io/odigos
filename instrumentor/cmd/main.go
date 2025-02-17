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

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"

	"github.com/odigos-io/odigos/instrumentor/controllers"
	"github.com/odigos-io/odigos/instrumentor/sdks"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	"github.com/odigos-io/odigos/instrumentor"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	_ "net/http/pprof"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	managerOptions := controllers.KubeManagerOptions{}
	var telemetryDisabled bool

	flag.StringVar(&managerOptions.MetricsServerBindAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&managerOptions.HealthProbeBindAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&managerOptions.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&telemetryDisabled, "telemetry-disabled", false, "Disable telemetry")

	opts := ctrlzap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zapLogger := ctrlzap.NewRaw(ctrlzap.UseFlagOptions(&opts))
	zapLogger = bridge.AttachToZapLogger(zapLogger)
	logger := zapr.NewLogger(zapLogger)
	managerOptions.Logger = logger

	// TODO: remove once the webhook stops using the default SDKs from the sdks package
	sdks.SetDefaultSDKs()

	// TODO: remove once we create an enterprise instrumentor
	tier := env.GetOdigosTierFromEnv()
	var defaulter distros.Defaulter
	switch tier {
	case common.CommunityOdigosTier:
		defaulter = distros.NewCommunityDefaulter()
	case common.OnPremOdigosTier:
		defaulter = distros.NewOnPremDefaulter()
	default:
		setupLog.Error(nil, "Invalid tier", "tier", tier)
		os.Exit(1)
	}

	distrosGetter, err := distros.NewGetter()
	if err != nil {
		setupLog.Error(err, "Failed to initialize distro getter")
		os.Exit(1)
	}
	dp, err := distros.NewProvider(defaulter, distrosGetter)
	if err != nil {
		setupLog.Error(err, "Failed to initialize distro provider")
		os.Exit(1)
	}

	i, err := instrumentor.New(managerOptions, dp)
	if err != nil {
		setupLog.Error(err, "Failed to initialize instrumentor")
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()

	i.Run(ctx, telemetryDisabled)
	logger.V(0).Info("instrumentor exiting")
}
