# OAuth2 Client Credentials Implementation for OTLP Destinations

## ✅ Implementation Complete

This document summarizes the successful implementation of OAuth2 client credentials authentication for both OTLP HTTP and OTLP gRPC destinations in Odigos.

## 🎯 Features Implemented

### 1. **UI Configuration**
- **Enable OAuth2** - Checkbox to enable/disable OAuth2 (follows TLS pattern)
- **Client ID** - OAuth2 client identifier (required)
- **Client Secret** - OAuth2 client secret (required, marked as secret)
- **Token URL** - OAuth2 token endpoint (required)
- **Scopes** - Comma-separated OAuth2 scopes (optional)
- **Audience** - OAuth2 audience parameter (optional)

### 2. **Backend Implementation**
- OAuth2 takes precedence over Basic Auth for HTTP destinations
- Proper error handling and validation
- Environment variable references for secrets
- Support for all OAuth2 client credentials parameters

### 3. **Collector Integration**
- Added `oauth2clientauthextension` to collector builder configuration
- Generated proper OpenTelemetry Collector configuration
- Full integration with existing collector architecture

## 📋 Files Modified

### UI Configuration Files
- **`destinations/data/otlphttp.yaml`** - Added OAuth2 UI fields for HTTP
- **`destinations/data/otlp.yaml`** - Added OAuth2 UI fields for gRPC

### Backend Configuration Files
- **`common/config/otlphttp.go`** - OAuth2 implementation for HTTP
- **`common/config/genericotlp.go`** - OAuth2 implementation for gRPC
- **`collector/builder-config.yaml`** - Added OAuth2 extension to collector

### Test Files
- **`common/config/otlphttp_test.go`** - Comprehensive OAuth2 tests

## 🔧 Configuration Parameters

### OTLP HTTP OAuth2 Fields
```yaml
OTLP_HTTP_OAUTH2_ENABLED: "true"           # Enable/disable OAuth2
OTLP_HTTP_OAUTH2_CLIENT_ID: "client-id"    # Client identifier
OTLP_HTTP_OAUTH2_CLIENT_SECRET: "secret"   # Client secret
OTLP_HTTP_OAUTH2_TOKEN_URL: "https://..."  # Token endpoint
OTLP_HTTP_OAUTH2_SCOPES: "api.metrics"     # Comma-separated scopes
OTLP_HTTP_OAUTH2_AUDIENCE: "api.service"   # Audience parameter
```

### OTLP gRPC OAuth2 Fields
```yaml
OTLP_GRPC_OAUTH2_ENABLED: "true"           # Enable/disable OAuth2
OTLP_GRPC_OAUTH2_CLIENT_ID: "client-id"    # Client identifier
OTLP_GRPC_OAUTH2_CLIENT_SECRET: "secret"   # Client secret
OTLP_GRPC_OAUTH2_TOKEN_URL: "https://..."  # Token endpoint
OTLP_GRPC_OAUTH2_SCOPES: "api.metrics"     # Comma-separated scopes
OTLP_GRPC_OAUTH2_AUDIENCE: "api.service"   # Audience parameter
```

## 🏗️ Generated Collector Configuration

The implementation generates proper OpenTelemetry Collector configuration:

### OAuth2 Extension Configuration
```yaml
extensions:
  oauth2client/otlphttp-{id}:
    client_id: your-client-id
    client_secret: ${OTLP_HTTP_OAUTH2_CLIENT_SECRET}
    token_url: https://auth.example.com/oauth2/token
    endpoint_params:
      audience: api.example.com
    scopes: ["api.metrics", "api.traces"]
```

### Exporter Configuration
```yaml
exporters:
  otlphttp/generic-{id}:
    endpoint: https://api.example.com:4318
    auth:
      authenticator: oauth2client/otlphttp-{id}
    tls:
      insecure: false
```

### Service Configuration
```yaml
service:
  extensions: [oauth2client/otlphttp-{id}]
  pipelines:
    traces:
      exporters: [otlphttp/generic-{id}]
    metrics:
      exporters: [otlphttp/generic-{id}]
    logs:
      exporters: [otlphttp/generic-{id}]
```

## 🚀 Usage Examples

### Example 1: OTLP HTTP with OAuth2
```yaml
# UI Configuration
Enable OAuth2: ✓
Client ID: my-client-id
Client Secret: my-client-secret
Token URL: https://auth.provider.com/oauth2/token
Scopes: api.metrics,api.traces
Audience: telemetry-api
```

### Example 2: OTLP gRPC with OAuth2
```yaml
# UI Configuration
Enable OAuth2: ✓
Client ID: grpc-client-id
Client Secret: grpc-client-secret
Token URL: https://auth.provider.com/oauth2/token
Scopes: grpc.api
```

## 🔒 Security Features

1. **Client secrets** are referenced via environment variables (`${OTLP_HTTP_OAUTH2_CLIENT_SECRET}`)
2. **UI marks secret fields** appropriately in the configuration
3. **OAuth2 takes precedence** over Basic Auth when both are configured
4. **Proper validation** ensures required fields are present

## ✅ Validation & Testing

### Test Coverage
- ✅ OAuth2 enabled with all parameters
- ✅ OAuth2 disabled (fallback behavior)
- ✅ OAuth2 enabled but missing required fields
- ✅ Basic Auth precedence when OAuth2 disabled
- ✅ OAuth2 precedence over Basic Auth
- ✅ Proper collector configuration generation
- ✅ Extension registration and service configuration

### Commands to Test
```bash
# Build collector with OAuth2 extension
cd /workspace/collector && make genodigoscol

# Run configuration tests
cd /workspace/common/config && go test -v -run TestOAuth2Configuration

# Run all tests
cd /workspace/common/config && go test -v
```

## 🎉 Result

The odigos-gateway ConfigMap will now properly contain:
1. **OAuth2 extension configuration** with client credentials parameters
2. **Exporter OAuth authentication** configuration
3. **Service extensions** registration
4. **Pipeline integration** for traces, metrics, and logs

Users can now configure OAuth2 client credentials authentication for both OTLP HTTP and OTLP gRPC destinations through the Odigos UI, and the collector will automatically handle OAuth2 token acquisition and renewal using the [OAuth2 Client Auth Extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/extension/oauth2clientauthextension/README.md).

## 📚 References

- [OAuth2 Client Auth Extension Documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/extension/oauth2clientauthextension/README.md)
- [OpenTelemetry Collector Configuration](https://opentelemetry.io/docs/collector/configuration/)
- [Building Custom Authenticator Extensions](https://opentelemetry.io/docs/collector/custom-auth/)