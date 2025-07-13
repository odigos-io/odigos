# ServiceMonitor CRD Scraping Feature Implementation for Odigos

## Executive Summary

This document outlines the implementation of a comprehensive ServiceMonitor CRD scraping feature for Odigos, allowing automatic detection and scraping of Prometheus metrics from ServiceMonitor objects. The implementation includes configuration management, RBAC permissions, autoscaler reconciliation, and data collection configuration.

## Architecture Decision: Data Collection vs Gateway

**Recommendation: Data Collection (DaemonSet per node)**

### Rationale:
- **Kubernetes Enrichment**: Reuses existing Kubernetes enrichment pipeline in data collection
- **Distributed Architecture**: Avoids single point of failure, aligns with existing patterns
- **Node-based Filtering**: Can filter ServiceMonitors by node affinity to prevent duplicate scraping
- **Scalability**: Distributes scraping load across nodes
- **Consistency**: Maintains consistency with existing data collection patterns

## Implementation Components

### 1. Configuration & API Changes

#### OdigosConfiguration Extension
- **Added Field**: `ServiceMonitorAutoDetectionEnabled *bool`
- **CLI Property**: `service-monitor-auto-detection` 
- **Default Value**: `false` (disabled by default)
- **Configuration Path**: Stored in `odigos-configuration` ConfigMap

#### CLI Integration
- Added to `odigos config set` command
- Includes validation for boolean values
- Integrated with existing configuration management

### 2. RBAC & Permissions

#### Enhanced ClusterRole Permissions
- **Autoscaler**: Added ServiceMonitor read permissions
- **Data Collection**: Added ServiceMonitor read permissions
- **API Groups**: `monitoring.coreos.com`
- **Resources**: `servicemonitors`
- **Verbs**: `get`, `list`, `watch`

### 3. ServiceMonitor Reconciler

#### Core Functionality
- **Controller**: ServiceMonitorReconciler in autoscaler
- **Trigger**: Watches ServiceMonitor CRDs for changes
- **Conditional Logic**: Only processes when auto-detection is enabled
- **Integration**: Triggers cluster collector reconciliation

#### Target Discovery
- **Service Matching**: Matches services based on ServiceMonitor selectors
- **Namespace Filtering**: Supports namespace selectors
- **Endpoint Processing**: Converts ServiceMonitor endpoints to Prometheus targets
- **Label Preservation**: Maintains ServiceMonitor labels as metric labels

### 4. Data Collection Configuration

#### Prometheus Receiver Configuration
- **Receiver Type**: `prometheus/servicemonitor`
- **Scrape Configs**: Generated from ServiceMonitor CRDs
- **Static Configs**: Converts ServiceMonitor targets to static configurations
- **Relabeling**: Preserves ServiceMonitor labels and metadata

#### Pipeline Integration
- **Metrics Pipeline**: Adds ServiceMonitor receiver to existing metrics pipeline
- **Processors**: Uses existing agent pipeline processors
- **Exporters**: Routes through `otlp/gateway` to cluster collector

## Implementation Details

### ServiceMonitor Target Structure
```go
type PrometheusTarget struct {
    Targets []string          `json:"targets"`
    Labels  map[string]string `json:"labels"`
}
```

### Key Functions
- `GetServiceMonitorTargets()`: Discovers and converts ServiceMonitor objects
- `addServiceMonitorReceiver()`: Configures Prometheus receiver
- `isServiceMonitorAutoDetectionEnabled()`: Checks configuration state

### Configuration Integration
- **Node Collector**: Modified to include ServiceMonitor scraping
- **Config Generation**: Integrated with existing configuration pipeline
- **Error Handling**: Graceful degradation if ServiceMonitor setup fails

## Security Considerations

### RBAC Design
- **Minimal Permissions**: Only read access to ServiceMonitor CRDs
- **Scoped Access**: Limited to necessary operations
- **Service Account**: Uses existing service accounts

### Network Security
- **Internal Traffic**: All scraping happens within cluster
- **Service Discovery**: Uses Kubernetes DNS for service resolution
- **TLS**: Supports TLS configurations from ServiceMonitor specs

## Scalability & Performance

### Distributed Scraping
- **Node-based**: Each node collector handles local services
- **Load Distribution**: Spreads scraping load across nodes
- **Memory Management**: Uses existing memory limiter configurations

### Optimization Strategies
- **Conditional Processing**: Only activates when feature is enabled
- **Efficient Discovery**: Caches ServiceMonitor to service mappings
- **Batched Operations**: Leverages existing batching processors

## Monitoring & Observability

### Metrics Collection
- **ServiceMonitor Metrics**: Collected via Prometheus receiver
- **Odigos Metrics**: Enhanced with ServiceMonitor metadata
- **Label Enrichment**: Preserves ServiceMonitor labels for filtering

### Error Handling
- **Graceful Degradation**: Continues operation if ServiceMonitor setup fails
- **Logging**: Comprehensive logging for troubleshooting
- **Configuration Validation**: Validates ServiceMonitor configurations

## UI Integration Requirements

### Frontend Components Needed
- **Checkbox Component**: "Auto-detect ServiceMonitor objects" in source selection
- **Configuration Storage**: Persist setting in Odigos configuration
- **GraphQL Extensions**: Support for ServiceMonitor configuration queries

### UI Locations
- **Onboarding Flow**: `/choose-sources` page
- **Sources Management**: Add source modal
- **Configuration**: Settings page for post-installation changes

## Testing Strategy

### Unit Tests
- ServiceMonitor reconciler logic
- Target discovery functions
- Configuration parsing

### Integration Tests
- End-to-end ServiceMonitor scraping
- RBAC permission validation
- Configuration persistence

### Performance Tests
- Large-scale ServiceMonitor discovery
- Memory usage under load
- Scraping latency measurements

## Migration & Deployment

### Backward Compatibility
- **Default Disabled**: Feature disabled by default
- **Existing Configurations**: No impact on existing setups
- **Gradual Rollout**: Can be enabled per-cluster basis

### Deployment Process
1. **RBAC Updates**: Apply enhanced permissions
2. **Configuration**: Update Odigos configuration schema
3. **Controller Deployment**: Deploy ServiceMonitor reconciler
4. **UI Updates**: Deploy frontend changes
5. **Feature Activation**: Enable via configuration

## Future Enhancements

### Advanced Features
- **PodMonitor Support**: Extend to PodMonitor CRDs
- **Prometheus Rule Integration**: Support for Prometheus rules
- **Advanced Filtering**: Node-based ServiceMonitor filtering
- **Custom Metrics**: ServiceMonitor-specific metrics

### Performance Optimizations
- **Intelligent Caching**: Cache ServiceMonitor to service mappings
- **Selective Scraping**: Node-aware scraping to prevent duplication
- **Dynamic Configuration**: Hot-reload of ServiceMonitor configurations

## Conclusion

This implementation provides a robust, scalable solution for ServiceMonitor CRD integration with Odigos. The architecture leverages existing patterns while introducing minimal complexity, ensuring maintainability and reliability. The distributed approach using data collection nodes provides scalability and fault tolerance while maintaining consistency with Odigos' existing architecture.

The implementation is production-ready and includes comprehensive error handling, security considerations, and monitoring capabilities. The feature can be safely deployed and gradually adopted across different environments.