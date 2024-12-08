package sdkconfig

import (
	"context"
	"slices"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SdkConfigManager struct {
	logger   logr.Logger
	mgr      ctrl.Manager
	odigosNs string
}

func NewSdkConfigManager(logger logr.Logger, mgr ctrl.Manager, connectionCache *connection.ConnectionsCache, odigosNs string) *SdkConfigManager {

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
		logger.Error(err, "unable to create controller for opamp server sdk config", "controller", "InstrumentationConfig")
	}

	// setup the controller to watch for changes in the collectors group CRD for node collector to recalculate enabled signals
	if err := (&CollectorsGroupReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		ConnectionCache: connectionCache,
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller for opamp server sdk config", "controller", "Destination")
	}

	return sdkConfigManager
}

func (m *SdkConfigManager) GetFullConfig(ctx context.Context, remoteResourceAttributes []configresolvers.ResourceAttribute, podWorkload *workload.PodWorkload, instrumentedAppName string, programmingLanguage string,
	instrumentationConfig *odigosv1.InstrumentationConfig) (*protobufs.AgentRemoteConfig, error) {

	var nodeCollectorGroup odigosv1.CollectorsGroup
	err := m.mgr.GetClient().Get(ctx, client.ObjectKey{Name: k8sconsts.OdigosNodeCollectorCollectorGroupName, Namespace: m.odigosNs}, &nodeCollectorGroup)
	if err != nil {
		return nil, err
	}
	tracesEnabled := slices.Contains(nodeCollectorGroup.Status.ReceiverSignals, common.TracesObservabilitySignal)

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
