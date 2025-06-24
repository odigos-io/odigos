# gRPC OAuth2 TLS Auto-Enable Fix

## 🛠️ Issue Resolved

**Error:** `grpc: the credentials require transport level security (use grpc.WithTransportCredentials() to set)`

This error occurred when OAuth2 authentication was enabled for gRPC OTLP exporters but TLS was disabled. gRPC requires transport-level security (TLS) when using any authentication credentials.

## ✅ Root Cause

gRPC has a security requirement: **when using authentication credentials (like OAuth2), the connection MUST be secured with TLS**. This is a fundamental gRPC security constraint.

The issue was that users could:
1. Enable OAuth2 authentication: `OTLP_GRPC_OAUTH2_ENABLED: "true"`
2. Disable TLS: `OTLP_GRPC_TLS_ENABLED: "false"`

This configuration is invalid and causes the collector to fail with the transport security error.

## 🔧 Fix Applied

**Auto-Enable TLS for OAuth2:**

```go
// Check for OAuth2 authentication early to determine TLS requirements
oauth2ExtensionName, oauth2ExtensionConf := applyGrpcOAuth2Auth(dest)
oauth2Enabled := oauth2ExtensionName != ""

// gRPC requires TLS when using authentication credentials like OAuth2
if oauth2Enabled && !tlsEnabled {
    tlsEnabled = true
}
```

**Logic:**
- If OAuth2 is enabled AND TLS is disabled → **automatically enable TLS**
- If OAuth2 is enabled AND TLS is already enabled → **keep TLS enabled**
- If OAuth2 is disabled → **respect user's TLS setting**

## 📋 Behavior Matrix

| OAuth2 Enabled | User TLS Setting | Final TLS Setting | Reason |
|----------------|------------------|-------------------|---------|
| ✅ Yes | ❌ Disabled | ✅ **Auto-Enabled** | gRPC security requirement |
| ✅ Yes | ✅ Enabled | ✅ Enabled | User preference respected |
| ❌ No | ❌ Disabled | ❌ Disabled | User preference respected |
| ❌ No | ✅ Enabled | ✅ Enabled | User preference respected |

## 🔄 Expected Configuration

**Before Fix (Failed):**
```yaml
exporters:
  otlp/generic-destination-id:
    endpoint: grpc://example.com:4317
    auth:
      authenticator: oauth2client/otlpgrpc-destination-id
    tls:
      insecure: true  # ❌ This caused the error!
```

**After Fix (Working):**
```yaml
exporters:
  otlp/generic-destination-id:
    endpoint: grpc://example.com:4317
    auth:
      authenticator: oauth2client/otlpgrpc-destination-id
    tls:
      insecure: false  # ✅ Auto-enabled for OAuth2 security
```

## ✅ Test Coverage

The fix includes comprehensive test coverage:

- ✅ **OAuth2 + TLS Disabled → Auto-Enable TLS**
- ✅ **OAuth2 + TLS Enabled → Keep TLS Enabled**  
- ✅ **No OAuth2 + TLS Disabled → Keep TLS Disabled**
- ✅ **No OAuth2 + TLS Enabled → Keep TLS Enabled**

## 🎯 Impact

**For Users:**
- OAuth2 authentication now works seamlessly with gRPC OTLP exporters
- No need to manually enable TLS when using OAuth2
- Prevents confusing transport security errors

**For Developers:**
- Automatic security compliance with gRPC requirements
- Maintains backward compatibility for non-OAuth2 scenarios
- Clear test coverage for all combinations

## 🚀 Resolution Steps

1. **Apply the fix** to `common/config/genericotlp.go`
2. **Restart your Odigos controller** to pick up the new logic
3. **Update or recreate** your gRPC OTLP destination with OAuth2
4. **Verify** the collector starts successfully without transport security errors

The collector will now automatically enable TLS when OAuth2 is configured for gRPC exporters, resolving the `grpc: the credentials require transport level security` error.

## 📚 Technical Details

**Files Modified:**
- `common/config/genericotlp.go` - Added auto-TLS logic
- `common/config/genericotlp_test.go` - Added comprehensive tests

**gRPC Security Requirement:**
When using gRPC with credentials (OAuth2, basic auth, etc.), the underlying transport must be secured with TLS. This is enforced by the gRPC library and cannot be bypassed.

**OpenTelemetry Collector:**
The OTLP gRPC exporter respects this gRPC security requirement, which is why the error occurred when OAuth2 was combined with insecure transport.