# Odigos Configuration

This directory contains the Odigos configuration for automatic telemetry collection and routing to ClickHouse.

## Files

- `clickhouse-destination.yaml` - Odigos Destination manifest with dynamic EC2 IP injection
- `sources.yaml` - Odigos Sources manifest for workload generators (auto-detected by Odigos)

## Configuration

The ClickHouse destination is automatically configured with:

- **Endpoint**: `tcp://<EC2_IP>:9000` (dynamically injected)
- **Database**: `otel`
- **Username**: `default`
- **Password**: `stresstest` (from EC2 user data script)
- **Tables**: Pre-configured for traces, metrics, and logs

## Deployment

The destination is automatically deployed as part of the Terraform EKS provisioning process:

1. **Step 1**: Deploy EKS infrastructure
2. **Step 2**: Deploy EC2 monitoring stack  
3. **Step 3**: Deploy Prometheus agent
4. **Step 4**: Deploy Odigos
5. **Step 5**: Deploy workload generators (only with `--with-load-test` flag)
6. **Step 6**: Apply Odigos sources (only with `--with-load-test` flag)
7. **Step 7**: Deploy Odigos ClickHouse destination with dynamic EC2 IP

### Conditional Deployment

- **Full deployment** (`./deploy.sh deploy`): Deploys everything including workload generators and Odigos sources
- **Core deployment** (`./deploy.sh k8s-apps`): Deploys Odigos and ClickHouse destination only
- **Load test deployment** (`./deploy.sh k8s-apps --with-load-test`): Deploys everything including workload generators and Odigos sources

### Current Status
**Working**: ClickHouse destination is successfully configured and processing telemetry data from workload generators.

## Automatic Source Detection

Odigos automatically detects and instruments applications with the `odigos-target=true` label:

```bash
# Check detected sources
kubectl get sources -n load-test

# Check source details
kubectl describe source <source-name> -n load-test

# Verify instrumentation
kubectl get pods -n load-test --show-labels | grep odigos-target
```

## Verification

After deployment, verify the destination is working:

```bash
# Check destination status
kubectl get destinations -n odigos-system

# Check destination details (should show "Destination successfully transformed to otelcol configuration")
kubectl describe destination clickhouse-destination -n odigos-system

# Check Odigos gateway logs (should show "Everything is ready. Begin running and processing data")
kubectl logs -l app.kubernetes.io/name=odigos -n odigos-system

# Check if data is flowing to ClickHouse
kubectl logs -l app.kubernetes.io/name=odigos -n odigos-system | grep -i clickhouse
```

## Tables Created

The destination automatically creates the following tables in ClickHouse:

- `otel_traces` - OpenTelemetry traces
- `otel_logs` - Application logs
- `otel_metrics_gauge` - Gauge metrics
- `otel_metrics_sum` - Sum metrics
- `otel_metrics_histogram` - Histogram metrics
- `otel_metrics_exponential_histogram` - Exponential histogram metrics
- `otel_metrics_summary` - Summary metrics

## Troubleshooting

### Destination Not Ready

```bash
# Check destination conditions
kubectl describe destination clickhouse-destination -n odigos-system

# Check secret exists
kubectl get secret clickhouse-secret -n odigos-system

# Check ClickHouse connectivity from EKS
kubectl run test-clickhouse --image=busybox --rm -it --restart=Never -- sh -c "nc -zv <EC2_IP> 9000"
```

### Connection Issues

```bash
# Verify ClickHouse is running on EC2
aws ssm start-session --target <EC2_INSTANCE_ID>
systemctl status clickhouse-server
ss -tlnp | grep 9000

# Check security groups
aws ec2 describe-security-groups --group-ids <EC2_SG_ID>
```

## Customization

To modify the ClickHouse configuration:

1. Edit `clickhouse-destination.yaml`
2. Re-run `terraform apply` to update the destination
3. The destination will be automatically updated in the cluster

## Data Flow

```
EKS Workload Generators → Odigos Sources → Odigos Gateway → ClickHouse (EC2)
                                                           ↓
                                                      Prometheus (EC2)
                                                           ↓
                                                      Grafana (EC2)
```

### Telemetry Types
- **Traces**: Request flows and spans
- **Metrics**: Application and system metrics  
- **Logs**: Application logs and events

All telemetry data is automatically collected by Odigos and routed to ClickHouse for storage and analysis.

