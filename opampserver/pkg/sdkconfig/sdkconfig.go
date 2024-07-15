package sdkconfig

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SdkConfigManager struct {
	logger logr.Logger
	mgr    ctrl.Manager
}

func NewSdkConfigManager(logger logr.Logger, mgr ctrl.Manager, connectionCache *connection.ConnectionsCache) *SdkConfigManager {

	sdkConfigManager := &SdkConfigManager{
		logger: logger,
		mgr:    mgr,
	}

	// setup the controller to watch for changes in the instrumentation configs CRD
	if err := (&InstrumentationConfigReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		ConnectionCache: connectionCache,
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller for opamp server sdk config", "controller", "InstrumentationConfig")
	}

	// setup the controller to watch for changes in the destinations CRD to recalculate enabled signals
	if err := (&DestinationReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		ConnectionCache: connectionCache,
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller for opamp server sdk config", "controller", "Destination")
	}

	return sdkConfigManager
}

func (m *SdkConfigManager) GetFullConfig(ctx context.Context, remoteResourceAttributes []configresolvers.ResourceAttribute, podWorkload *common.PodWorkload, instrumentedAppName string) (*protobufs.AgentRemoteConfig, error) {

	// query which signals are enabled in the current destinations list
	tracesEnabled, _, err := configresolvers.CalcEnabledSignals(ctx, m.mgr.GetClient())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate enabled signals: %w", err)
	}

	sdkRemoteConfig := configsections.CalcSdkRemoteConfig(remoteResourceAttributes, tracesEnabled)
	opampRemoteConfigSdk, sdkSectionName, err := configsections.SdkRemoteConfigToOpamp(sdkRemoteConfig)
	if err != nil {
		m.logger.Error(err, "failed to marshal server offered resource attributes")
		return nil, err
	}

	instrumentationLibrariesRemoteConfig, err := configsections.CalcInstrumentationLibrariesRemoteConfig(ctx, m.mgr.GetClient(), instrumentedAppName, podWorkload.Namespace)
	if err != nil {
		m.logger.Error(err, "failed to calculate instrumentation libraries config", "k8sAttributes", remoteResourceAttributes)
		return nil, err
	}

	opampRemoteConfigInstrumentationLibraries, instrumentationLibrariesSectionName, err := configsections.InstrumentationLibrariesRemoteConfigToOpamp(instrumentationLibrariesRemoteConfig)
	if err != nil {
		m.logger.Error(err, "failed to marshal instrumentation libraries config")
		return nil, err
	}

	agentConfigMap := protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			sdkSectionName:                      opampRemoteConfigSdk,
			instrumentationLibrariesSectionName: opampRemoteConfigInstrumentationLibraries,
		},
	}
	configHash := connection.CalcRemoteConfigHash(&agentConfigMap)
	serverAttrsRemoteCfg := protobufs.AgentRemoteConfig{
		Config:     &agentConfigMap,
		ConfigHash: configHash,
	}

	return &serverAttrsRemoteCfg, nil
}
