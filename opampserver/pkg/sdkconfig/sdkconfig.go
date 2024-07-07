package sdkconfig

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SdkConfigManager struct {
	logger   logr.Logger
	mgr      ctrl.Manager
	nodeName string
}

func NewSdkConfigManager(logger logr.Logger, mgr ctrl.Manager, nodeName string) *SdkConfigManager {
	return &SdkConfigManager{
		logger:   logger,
		mgr:      mgr,
		nodeName: nodeName,
	}
}

func (m *SdkConfigManager) GetFullConfig(ctx context.Context, k8sAttributes *deviceid.K8sResourceAttributes, instrumentationLibraryNames []string) (*protobufs.AgentRemoteConfig, error) {

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
	instrumentationSdkConfig := &v1alpha1.InstrumentationConfig{}
	err = m.mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: k8sAttributes.Namespace, Name: configObjectName}, instrumentationSdkConfig)
	if err != nil && !apierrors.IsNotFound(err) {
		m.logger.Error(err, "failed to get instrumentation sdk config", "configObjectName", configObjectName)
		return nil, err
	}

	instrumentationLibrariesConfig := make([]RemoteConfigInstrumentationLibrary, 0)
	for _, instrumentationLibraryName := range instrumentationLibraryNames {
		// find if there is any config for this instrumentation in crd
		var instrumentationLibCrdConfig *v1alpha1.InstrumentationLibraryConfig
		for _, sdkConfig := range instrumentationSdkConfig.Spec.SdkConfigs {
			for _, instrumentationConfig := range sdkConfig.InstrumentationLibraryConfigs {
				if instrumentationConfig.InstrumentationLibraryName == instrumentationLibraryName {
					instrumentationLibCrdConfig = &instrumentationConfig
					break
				}
			}
		}

		tracesDisabled := false // enabled by default, unless explicitly disabled
		if instrumentationLibCrdConfig != nil {
			// if we found config, use it
			if instrumentationLibCrdConfig.TraceConfig != nil {
				if instrumentationLibCrdConfig.TraceConfig.Disabled != nil {
					tracesDisabled = *instrumentationLibCrdConfig.TraceConfig.Disabled
				}
			}
		}
		instrumentationLibrariesConfig = append(instrumentationLibrariesConfig, RemoteConfigInstrumentationLibrary{
			Name: instrumentationLibraryName,
			Traces: RemoteConfigInstrumentationLibraryTraces{
				Disabled: tracesDisabled,
			},
		})
	}

	instrumentationLibrariesConfigBytes, err := json.Marshal(instrumentationLibrariesConfig)
	if err != nil {
		m.logger.Error(err, "failed to marshal instrumentation libraries config")
		return nil, err
	}

	serverAttrsRemoteCfg := protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
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
		},
	}

	return &serverAttrsRemoteCfg, nil
}
