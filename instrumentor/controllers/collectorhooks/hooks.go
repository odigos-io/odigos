package collectorhooks

import (
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/instrumentor/controllers/nodecollector/collectorconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// CollectorHooks is the enterprise extension point for collector ConfigMaps
// and the cluster-collector Deployment. Nil, or nil fields, means community.
type CollectorHooks struct {
	NodeCollectorConfig    NodeCollectorConfig
	ClusterCollectorConfig ClusterCollectorConfig
}

type NodeCollectorConfig interface {
	MutateConfig(NodeCollectorHookInput) error
}

type ClusterCollectorConfig interface {
	MutateConfig(GatewayHookInput) error
	MutateDeployment(*GatewayDeploymentHookInput)
}

type configMutator struct {
	Config *config.Config
}

func (m *configMutator) ensureConfig() {
	if m.Config == nil {
		m.Config = &config.Config{}
	}
}

func (m *configMutator) AddReceiver(name string, cfg config.GenericMap) {
	m.ensureConfig()
	if m.Config.Receivers == nil {
		m.Config.Receivers = config.GenericMap{}
	}
	m.Config.Receivers[name] = cfg
}

func (m *configMutator) AddProcessor(name string, cfg config.GenericMap) {
	m.ensureConfig()
	if m.Config.Processors == nil {
		m.Config.Processors = config.GenericMap{}
	}
	m.Config.Processors[name] = cfg
}

func (m *configMutator) AddExporter(name string, cfg config.GenericMap) {
	m.ensureConfig()
	if m.Config.Exporters == nil {
		m.Config.Exporters = config.GenericMap{}
	}
	m.Config.Exporters[name] = cfg
}

func (m *configMutator) AddExtension(name string, cfg config.GenericMap) {
	m.ensureConfig()
	if m.Config.Extensions == nil {
		m.Config.Extensions = config.GenericMap{}
	}
	m.Config.Extensions[name] = cfg
	if !slices.Contains(m.Config.Service.Extensions, name) {
		m.Config.Service.Extensions = append(m.Config.Service.Extensions, name)
	}
}

func (m *configMutator) AddPipeline(name string, p config.Pipeline) {
	m.ensureConfig()
	if m.Config.Service.Pipelines == nil {
		m.Config.Service.Pipelines = map[string]config.Pipeline{}
	}
	m.Config.Service.Pipelines[name] = p
}

func (m *configMutator) appendReceiver(pipelineName, receiverName string) {
	m.ensureConfig()
	if m.Config.Service.Pipelines == nil {
		return
	}
	p, ok := m.Config.Service.Pipelines[pipelineName]
	if !ok {
		return
	}
	p.Receivers = append(p.Receivers, receiverName)
	m.Config.Service.Pipelines[pipelineName] = p
}

func (m *configMutator) appendExporter(pipelineName, exporterName string) {
	m.ensureConfig()
	if m.Config.Service.Pipelines == nil {
		return
	}
	p, ok := m.Config.Service.Pipelines[pipelineName]
	if !ok {
		return
	}
	p.Exporters = append(p.Exporters, exporterName)
	m.Config.Service.Pipelines[pipelineName] = p
}

func (m *configMutator) setReceivers(pipelineName string, receivers []string) {
	m.ensureConfig()
	if m.Config.Service.Pipelines == nil {
		return
	}
	p, ok := m.Config.Service.Pipelines[pipelineName]
	if !ok {
		return
	}
	p.Receivers = receivers
	m.Config.Service.Pipelines[pipelineName] = p
}

// NodeCollectorHookInput is passed to NodeCollectorConfig.MutateConfig after
// OSS merges the community node-collector config.
type NodeCollectorHookInput struct {
	configMutator
	Sources            *odigosv1.InstrumentationConfigList
	Profiling          *common.ProfilingConfiguration
	ProfilesProcessors []string
	OdigosNamespace    string
}

func (in *NodeCollectorHookInput) AppendReceiverToTraces(name string) {
	in.appendReceiver(collectorconfig.OdigosTracesPipelineName, name)
}

func (in *NodeCollectorHookInput) SetLogsReceivers(names []string) {
	in.setReceivers(collectorconfig.LogsPipelineName, names)
}

// GatewayHookInput is passed to ClusterCollectorConfig.MutateConfig after
// pipelinegen and OSS self-telemetry.
type GatewayHookInput struct {
	configMutator
	Profiling          *common.ProfilingConfiguration
	Insights           *common.InsightsConfiguration
	CardinalityControl *common.CardinalityControlConfiguration
	OdigosNamespace    string
}

func (in *GatewayHookInput) AppendExporterToRootTraces(name string) {
	in.appendExporter(pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal), name)
}

func (in *GatewayHookInput) HasRootTracesPipeline() bool {
	in.ensureConfig()
	if in.Config.Service.Pipelines == nil {
		return false
	}
	_, ok := in.Config.Service.Pipelines[pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)]
	return ok
}

// GatewayDeploymentHookInput is passed to ClusterCollectorConfig.MutateDeployment
// after OSS builds the community cluster-collector Deployment.
type GatewayDeploymentHookInput struct {
	Deployment   *appsv1.Deployment
	Profiling    *common.ProfilingConfiguration
	featureGates []string
}

func (in *GatewayDeploymentHookInput) AddEnv(env corev1.EnvVar) {
	if in.Deployment == nil || len(in.Deployment.Spec.Template.Spec.Containers) == 0 {
		return
	}
	c := &in.Deployment.Spec.Template.Spec.Containers[0]
	c.Env = append(c.Env, env)
}

func (in *GatewayDeploymentHookInput) AddFeatureGate(name string) {
	in.featureGates = append(in.featureGates, name)
}

func (in *GatewayDeploymentHookInput) FeatureGates() []string {
	return in.featureGates
}

func ApplyNodeConfig(hooks *CollectorHooks, in NodeCollectorHookInput) error {
	if hooks == nil || hooks.NodeCollectorConfig == nil {
		return nil
	}
	return hooks.NodeCollectorConfig.MutateConfig(in)
}

func ApplyGatewayConfig(hooks *CollectorHooks, in GatewayHookInput) error {
	if !HasClusterCollectorConfig(hooks) {
		return nil
	}
	return hooks.ClusterCollectorConfig.MutateConfig(in)
}

func ApplyGatewayDeployment(hooks *CollectorHooks, in *GatewayDeploymentHookInput) {
	if !HasClusterCollectorConfig(hooks) {
		return
	}
	hooks.ClusterCollectorConfig.MutateDeployment(in)
}

// HasClusterCollectorConfig is true when enterprise will mutate the gateway.
// OSS uses this to pass Insights into pipelinegen (groupbytrace) only then.
func HasClusterCollectorConfig(hooks *CollectorHooks) bool {
	return hooks != nil && hooks.ClusterCollectorConfig != nil
}
