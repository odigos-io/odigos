package sdkconfig

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SdkConfigManager struct {
	logger   logr.Logger
	mgr      ctrl.Manager
	nodeName string
}

func NewSdkConfigManager(logger logr.Logger, mgr ctrl.Manager, connectionCache *connection.ConnectionsCache, nodeName string) *SdkConfigManager {

	sdkConfigManager := &SdkConfigManager{
		logger:   logger,
		mgr:      mgr,
		nodeName: nodeName,
	}

	// setup the controller to watch for changes in the instrumentation configs CRD
	if err := (&InstrumentationConfigReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		ConnectionCache: connectionCache,
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller for opamp server sdk config", "controller", "InstrumentationConfig")
	}

	return sdkConfigManager
}

func (m *SdkConfigManager) GetFullConfig(ctx context.Context, k8sAttributes *deviceid.K8sResourceAttributes) (*protobufs.AgentRemoteConfig, error) {

	serverOfferedResourceAttributes, err := calculateServerAttributes(k8sAttributes, m.nodeName)
	if err != nil {
		m.logger.Error(err, "failed to calculate server attributes", "k8sAttributes", k8sAttributes)
		return nil, err
	}

	remoteConfigSdkBytes, err := json.Marshal(RemoteConfigSdk{RemoteResourceAttributes: serverOfferedResourceAttributes})
	if err != nil {
		m.logger.Error(err, "failed to marshal server offered resource attributes")
		return nil, err
	}

	configObjectName := workload.GetRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)
	instrumentationLibrariesConfig, err := calcInstrumentationLibrariesRemoteConfig(ctx, m.mgr.GetClient(), configObjectName, k8sAttributes.Namespace)
	if err != nil {
		m.logger.Error(err, "failed to calculate instrumentation libraries config", "k8sAttributes", k8sAttributes)
		return nil, err
	}

	instrumentationLibrariesConfigBytes, err := json.Marshal(instrumentationLibrariesConfig)
	if err != nil {
		m.logger.Error(err, "failed to marshal instrumentation libraries config")
		return nil, err
	}

	agentConfigMap := protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			string(RemoteConfigSdkConfigSectionName): {
				Body:        remoteConfigSdkBytes,
				ContentType: "application/json",
			},
			string(RemoteConfigInstrumentationLibrariesConfigSectionName): {
				Body:        instrumentationLibrariesConfigBytes,
				ContentType: "application/json",
			},
		},
	}
	configHash := connection.CalcRemoteConfigHash(&agentConfigMap)
	serverAttrsRemoteCfg := protobufs.AgentRemoteConfig{
		Config:     &agentConfigMap,
		ConfigHash: configHash,
	}

	return &serverAttrsRemoteCfg, nil
}
