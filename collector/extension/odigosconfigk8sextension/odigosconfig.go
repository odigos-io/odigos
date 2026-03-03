package odigosconfigk8sextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"k8s.io/client-go/dynamic/dynamicinformer"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

// OdigosWorkloadConfig is an extension that runs a dynamic informer for InstrumentationConfigs
// and maintains a cache of workload sampling config keyed by WorkloadKey (namespace, kind, name).
type OdigosWorkloadConfig struct {
	cache           *cache
	logger          *zap.Logger
	cancel          context.CancelFunc
	informerFactory dynamicinformer.DynamicSharedInformerFactory // set when in-cluster; nil otherwise
}

// OdigosConfigExtension is the interface that must be implemented by an extension that wants to provide Odigos configuration.
var _ collector.OdigosConfigExtension = (*OdigosWorkloadConfig)(nil)

// NewOdigosConfig creates a new OdigosConfig extension.
func NewOdigosConfig(settings component.TelemetrySettings) (*OdigosWorkloadConfig, error) {
	return &OdigosWorkloadConfig{
		cache:  newCache(),
		logger: settings.Logger,
	}, nil
}

// Start starts the dynamic informer for InstrumentationConfigs. The informer
// fills the cache with workload sampling configs keyed by WorkloadKey.
func (o *OdigosWorkloadConfig) Start(ctx context.Context, _ component.Host) error {
	ctx, o.cancel = context.WithCancel(ctx)
	return o.startInformer(ctx)
}

// Shutdown stops the informer and clears the cache.
func (o *OdigosWorkloadConfig) Shutdown(ctx context.Context) error {
	if o.cancel != nil {
		o.cancel()
	}
	return nil
}

func (o *OdigosWorkloadConfig) GetFromResource(res pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	key, err := workloadKeyFromResourceAttributes(res.Attributes())
	if err != nil {
		return nil, false
	}
	return o.cache.Get(key)
}
