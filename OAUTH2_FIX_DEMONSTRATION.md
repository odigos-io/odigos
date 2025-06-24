# OAuth2 Fix Demonstration

## 🛠️ Issue Fixed

The OAuth2 client credentials configuration was not appearing in the odigos-gateway ConfigMap because the validation logic was incorrectly checking for the client secret in the regular config data instead of understanding that it's stored in the referenced secret.

## ✅ Root Cause

The issue was in these functions:
- `applyOAuth2Auth()` in `common/config/otlphttp.go`
- `applyGrpcOAuth2Auth()` in `common/config/genericotlp.go`

**Before Fix:**
```go
clientSecret := config[otlpHttpOAuth2ClientSecretKey]  // This was empty!
if clientId == "" || clientSecret == "" || tokenUrl == "" {
    return "", nil  // This caused early return!
}
```

**After Fix:**
```go
// Note: client secret is stored in the secret and injected as environment variable
// We don't validate it here since it's not in the regular config data
if clientId == "" || tokenUrl == "" {
    return "", nil
}
```

## 📝 Expected ConfigMap After Fix

With the user's destination configuration:
```yaml
spec:
  data:
    OTLP_HTTP_ENDPOINT: http://eden.com
    OTLP_HTTP_COMPRESSION: none
    OTLP_HTTP_TLS_ENABLED: "false"
    OTLP_HTTP_OAUTH2_ENABLED: "true"
    OTLP_HTTP_OAUTH2_CLIENT_ID: "123123"
    OTLP_HTTP_OAUTH2_TOKEN_URL: https://gooogle.com
    OTLP_HTTP_OAUTH2_SCOPES: asdasd
    OTLP_HTTP_OAUTH2_AUDIENCE: ccccx
  secretRef:
    name: odigos.io.dest.otlphttp-5s5jl  # Contains OTLP_HTTP_OAUTH2_CLIENT_SECRET
```

The ConfigMap should now include:

### Extensions Section:
```yaml
extensions:
  health_check:
    endpoint: 0.0.0.0:13133
  pprof:
    endpoint: 0.0.0.0:1777
  oauth2client/otlphttp-odigos.io.dest.otlphttp-8dfjl:
    client_id: "123123"
    client_secret: ${OTLP_HTTP_OAUTH2_CLIENT_SECRET}
    token_url: https://gooogle.com
    endpoint_params:
      audience: ccccx
    scopes: ["asdasd"]
```

### Exporters Section:
```yaml
exporters:
  otlphttp/generic-odigos.io.dest.otlphttp-8dfjl:
    compression: none
    endpoint: http://eden.com
    auth:
      authenticator: oauth2client/otlphttp-odigos.io.dest.otlphttp-8dfjl
    tls:
      insecure: true
      insecure_skip_verify: false
```

### Service Section:
```yaml
service:
  extensions:
  - health_check
  - pprof
  - oauth2client/otlphttp-odigos.io.dest.otlphttp-8dfjl
  pipelines:
    # ... existing pipelines with OAuth2 authenticator
```

## 🔄 How to Apply the Fix

1. **Restart the Odigos controller** to pick up the new configuration logic
2. **Trigger ConfigMap regeneration** by updating the destination or restarting the controller
3. **Verify the OAuth2 extension** appears in the ConfigMap

The collector will now:
1. Load the OAuth2 client auth extension
2. Use the client secret from the injected environment variable
3. Automatically handle OAuth2 token acquisition and renewal
4. Authenticate all requests to the OTLP HTTP endpoint using OAuth2

## ✅ Validation

The fix has been validated with:
- ✅ Unit tests covering all OAuth2 scenarios
- ✅ Test case matching the exact user configuration
- ✅ Verification that OAuth2 takes precedence over Basic Auth
- ✅ Proper secret handling through environment variables

## 🎉 Result

After applying this fix, users will see the OAuth2 extension and authentication configuration properly generated in the odigos-gateway ConfigMap, enabling OAuth2 client credentials authentication for OTLP destinations.