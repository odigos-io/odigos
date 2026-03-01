package sdkconfig

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SdkConfigManager struct {
	logger   *slog.Logger
	mgr      ctrl.Manager
	odigosNs string
}

func NewSdkConfigManager(mgr ctrl.Manager, connectionCache *connection.ConnectionsCache, odigosNs string) *SdkConfigManager {
	logger := commonlogger.Logger().With("subsystem", "sdkconfig")
	sdkConfigManager := &SdkConfigManager{
		logger:   logger,
		mgr:      mgr,
		odigosNs: odigosNs,
	}

	// setup the controller to watch for changes in the instrumentation configs CRD
	if err := (&InstrumentationConfigReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		ConnectionCache: connectionCache,
	}).SetupWithManager(mgr); err != nil {
		logger.Error("unable to create controller for opamp server sdk config", "err", err, "controller", "InstrumentationConfig")
	}

	return sdkConfigManager
}

func (m *SdkConfigManager) GetFullConfig(ctx context.Context, remoteResourceAttributes []configresolvers.ResourceAttribute, podWorkload *k8sconsts.PodWorkload, instrumentedAppName string, programmingLanguage string,
	instrumentationConfig *odigosv1.InstrumentationConfig, containerName string) (*protobufs.AgentRemoteConfig, error) {

	containerConfig := container.GetContainerConfigByName(instrumentationConfig.Spec.Containers, containerName)
	if containerConfig == nil {
		return nil, fmt.Errorf("container config not found for container %s", containerName)
	}

	sdkRemoteConfig := configsections.CalcSdkRemoteConfig(remoteResourceAttributes, containerConfig)
	opampRemoteConfigSdk, sdkSectionName, err := configsections.SdkRemoteConfigToOpamp(sdkRemoteConfig)
	if err != nil {
		m.logger.Error("failed to marshal server offered resource attributes", "err", err)
		return nil, err
	}

	instrumentationLibrariesRemoteConfig, err := configsections.CalcInstrumentationLibrariesRemoteConfig(ctx, m.mgr.GetClient(), instrumentedAppName, podWorkload.Namespace)
	if err != nil {
		m.logger.Error("failed to calculate instrumentation libraries config", "err", err, "k8sAttributes", remoteResourceAttributes)
		return nil, err
	}

	opampRemoteConfigInstrumentationLibraries, instrumentationLibrariesSectionName, err := configsections.InstrumentationLibrariesRemoteConfigToOpamp(instrumentationLibrariesRemoteConfig)
	if err != nil {
		m.logger.Error("failed to marshal instrumentation libraries config", "err", err)
		return nil, err
	}

	// // We are moving towards passing all Instrumentation capabilities unchanged within the instrumentationConfig to the opamp client.
	// // Gradually, we will migrate the InstrumentationLibraryConfigs and SDK remote config into the instrumentationConfig and the agents to use it.
	opampRemoteConfigInstrumentationConfig, err := configsections.FilterRelevantSdk(instrumentationConfig, programmingLanguage)
	if err != nil {
		m.logger.Error("failed to filter relevant sdk config", "err", err)
		return nil, err
	}
	opampRemoteConfigContainerConfig, err := configsections.FilterRelevantContainerConfig(instrumentationConfig, containerName)
	if err != nil {
		m.logger.Error("failed to filter relevant container config", "err", err)
		return nil, err
	}

	agentConfigMap := protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			sdkSectionName:                      opampRemoteConfigSdk,
			instrumentationLibrariesSectionName: opampRemoteConfigInstrumentationLibraries,
			"":                                  opampRemoteConfigInstrumentationConfig,
			"container_config":                  opampRemoteConfigContainerConfig,
		},
	}
	configHash := connection.CalcRemoteConfigHash(&agentConfigMap)
	serverAttrsRemoteCfg := protobufs.AgentRemoteConfig{
		Config:     &agentConfigMap,
		ConfigHash: configHash,
	}

	return &serverAttrsRemoteCfg, nil
}
