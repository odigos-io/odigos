# ServiceMonitor Auto-Detection CLI Usage Guide

This guide demonstrates how to configure and use the ServiceMonitor auto-detection feature in Odigos using the CLI.

## Overview

The ServiceMonitor auto-detection feature allows Odigos to automatically discover and scrape metrics from ServiceMonitor CRDs created by the Prometheus Operator. This enables seamless integration with existing Prometheus monitoring setups.

## Prerequisites

- Odigos installed in your Kubernetes cluster
- Optional: Prometheus Operator and ServiceMonitor CRDs (feature works safely without them)

## Configuration Commands

### Enable ServiceMonitor Auto-Detection

```bash
# Enable ServiceMonitor auto-detection
odigos config set service-monitor-auto-detection true
```

### Disable ServiceMonitor Auto-Detection

```bash
# Disable ServiceMonitor auto-detection (default)
odigos config set service-monitor-auto-detection false
```

### Check Current Configuration

```bash
# View all current configuration settings
odigos config
```

## Usage Examples

### Example 1: Basic Setup

```bash
# Step 1: Install Odigos (if not already installed)
odigos install

# Step 2: Enable ServiceMonitor auto-detection
odigos config set service-monitor-auto-detection true

# Step 3: Verify configuration
odigos config
```

### Example 2: Enable During Installation

```bash
# Install Odigos and immediately enable ServiceMonitor auto-detection
odigos install
odigos config set service-monitor-auto-detection true
```

### Example 3: Full Configuration with Other Settings

```bash
# Configure multiple settings including ServiceMonitor auto-detection
odigos config set telemetry-enabled true
odigos config set service-monitor-auto-detection true
odigos config set ignored-namespaces kube-system,kube-public
```

## How It Works

1. **Configuration**: When enabled, Odigos stores the setting in the `odigos-configuration` ConfigMap
2. **CRD Detection**: Odigos automatically detects if ServiceMonitor CRDs exist in the cluster
3. **Safe Operation**: If Prometheus Operator is not installed, the feature safely does nothing
4. **Metrics Collection**: When ServiceMonitors are found, Odigos automatically configures Prometheus receivers
5. **Data Flow**: Scraped metrics flow through the standard Odigos pipeline to your configured destinations

## Verification

### Check if Feature is Active

```bash
# Check Odigos configuration
kubectl get configmap odigos-configuration -n odigos-system -o yaml

# Look for: serviceMonitorAutoDetectionEnabled: true
```

### Check for ServiceMonitor CRDs

```bash
# Check if ServiceMonitor CRDs exist
kubectl get crd servicemonitors.monitoring.coreos.com

# List existing ServiceMonitors
kubectl get servicemonitors --all-namespaces
```

### Check Data Collection Configuration

```bash
# Check node collector configuration
kubectl get configmap odigos-data-collection -n odigos-system -o yaml

# Look for prometheus/servicemonitor receiver in the configuration
```

## Troubleshooting

### Feature Not Working

1. **Check Configuration**:
   ```bash
   odigos config
   # Verify service-monitor-auto-detection is set to true
   ```

2. **Check ServiceMonitor CRDs**:
   ```bash
   kubectl get crd servicemonitors.monitoring.coreos.com
   # Should exist if Prometheus Operator is installed
   ```

3. **Check Odigos Logs**:
   ```bash
   kubectl logs -n odigos-system deployment/odigos-autoscaler
   # Look for ServiceMonitor-related log messages
   ```

### No Metrics Being Scraped

1. **Verify ServiceMonitors Exist**:
   ```bash
   kubectl get servicemonitors --all-namespaces
   ```

2. **Check Service Labels**:
   ```bash
   # Ensure services have labels that match ServiceMonitor selectors
   kubectl get services --show-labels
   ```

3. **Check Data Collection Configuration**:
   ```bash
   kubectl get configmap odigos-data-collection -n odigos-system -o yaml
   # Look for prometheus/servicemonitor in receivers section
   ```

## Security Considerations

- The feature only requires read access to ServiceMonitor CRDs
- No additional network permissions are needed
- All scraping happens within the cluster using existing service discovery

## Performance Impact

- **Minimal Overhead**: Feature only activates when ServiceMonitors are present
- **Distributed Scraping**: Uses existing node collector architecture for scalability
- **Memory Efficient**: Leverages existing memory management and limits

## Migration from Prometheus

If you're migrating from a Prometheus setup:

1. **Keep Existing ServiceMonitors**: No changes needed to existing ServiceMonitor configurations
2. **Enable Feature**: `odigos config set service-monitor-auto-detection true`
3. **Configure Destinations**: Ensure Odigos destinations are configured for metrics
4. **Gradual Migration**: Can run alongside existing Prometheus instances

## Related Commands

```bash
# View all available configuration options
odigos config --help

# Set multiple configurations
odigos config set telemetry-enabled true
odigos config set ui-mode readonly
odigos config set service-monitor-auto-detection true

# Install with specific namespace
odigos install --namespace my-odigos-namespace
```

## Support

For issues or questions:
- Check the [Odigos documentation](https://docs.odigos.io)
- Review ServiceMonitor CRD compatibility
- Ensure Kubernetes RBAC permissions are properly configured