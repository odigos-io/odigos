# Environment Variables Test Suite

## Purpose
This test suite verifies that Odigos correctly detects, preserves, and modifies environment variables during instrumentation, ensuring that application-specific environment variables continue to work after instrumentation is applied.

## What is Tested

### Environment Variable Detection
- **Container runtime variables** - Environment variables set in Dockerfile
- **Manifest variables** - Environment variables defined in Kubernetes manifests
- **Variable preservation** - Ensuring existing variables remain functional
- **Variable modification** - Proper instrumentation variable injection

### Test Workloads

#### Node.js Environment Variables
- `nodejs-dockerfile-env` - NODE_OPTIONS set in Dockerfile
- `nodejs-manifest-env` - NODE_OPTIONS set in deployment manifest

#### Java Environment Variables
- `java-supported-docker-env` - Environment variables from container image
- `java-supported-manifest-env` - JAVA_TOOL_OPTIONS set in manifest

#### Python Environment Variables
- `python-alpine` - PYTHONPATH set in deployment manifest

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Verify installation → Start trace DB
```

### 2. Application Deployment
```
Deploy workloads with environment variables → Wait for pods to be ready
```

### 3. Environment Detection
```
Instrument namespace → Trigger runtime detection → Analyze environment variables
```

### 4. Validation
```
Assert environment variable detection → Verify instrumentation success
```

## Key Validations

### Environment Variable Detection Patterns

#### Container Runtime Detection (Dockerfile ENV)
For workloads with environment variables set in the container image:
- **Detection**: `envFromContainerRuntime` populated with detected variables
- **Example**: `nodejs-dockerfile-env` should detect NODE_OPTIONS from Dockerfile
- **Assertion**: `envFromContainerRuntime` contains expected name/value pairs

#### Manifest Variable Handling (Kubernetes ENV)
For workloads with environment variables in deployment manifest:
- **Detection**: `envFromContainerRuntime` should be null (not detected from runtime)
- **Example**: `nodejs-manifest-env` has NODE_OPTIONS in manifest, not runtime
- **Assertion**: `envFromContainerRuntime: null`

### Specific Environment Variable Tests

#### Node.js NODE_OPTIONS
- **Dockerfile ENV**: Detected from container runtime
  - Variable: `NODE_OPTIONS: "--require /app/execute_before.js --max-old-space-size=256"`
  - Source: Container image environment
- **Manifest ENV**: Not detected from runtime (present in manifest)
  - Variable: Same NODE_OPTIONS value
  - Source: Kubernetes deployment spec

#### Java JAVA_TOOL_OPTIONS
- **Dockerfile ENV**: Detected from container runtime
  - Variable: `LD_PRELOAD: ""`
  - Source: Container image environment
- **Manifest ENV**: Not detected from runtime
  - Variable: `JAVA_TOOL_OPTIONS: "-Dnot.work=true"`
  - Source: Kubernetes deployment spec

#### Python PYTHONPATH
- **Manifest ENV**: Not detected from runtime
  - Variable: `PYTHONPATH: "/app"`
  - Source: Kubernetes deployment spec

## Expected Behavior

### Detection Logic
1. **Runtime Scan** - Odigos scans container runtime for environment variables
2. **Manifest Check** - Variables in pod manifest are NOT included in runtime detection
3. **Preservation** - All environment variables (runtime + manifest) are preserved
4. **Instrumentation** - Additional instrumentation variables are injected alongside existing ones

### Variable Sources
- **Container Runtime** (`envFromContainerRuntime` populated)
  - Set via Dockerfile ENV commands
  - Available in container environment at runtime
  - Detected during runtime inspection

- **Kubernetes Manifest** (`envFromContainerRuntime: null`)
  - Set via deployment spec env section
  - Not detected during runtime scan
  - Handled separately by Kubernetes

## Files Structure
```
environment-variables/
├── README.md                          # This file
├── chainsaw-test.yaml                 # Main test definition
├── 01-install-test-apps.yaml          # Environment variable test workloads
└── 01-assert-env-vars.yaml           # Environment variable detection assertions
```

## Test Scenarios

### Scenario 1: Dockerfile Environment Variables
```yaml
# Container has: ENV NODE_OPTIONS="--require /app/execute_before.js"
# Expected: envFromContainerRuntime populated with NODE_OPTIONS
```

### Scenario 2: Manifest Environment Variables
```yaml
# Deployment has: env: [name: NODE_OPTIONS, value: "--require /app/execute_before.js"]
# Expected: envFromContainerRuntime: null (not detected from runtime)
```

### Scenario 3: Mixed Environment Variables
```yaml
# Container has some vars, manifest has others
# Expected: Only container vars in envFromContainerRuntime
```

## Dependencies
- Requires runtime detection capabilities
- Depends on environment variable scanning during runtime inspection
- Uses instrumentation device injection to preserve variables

## Duration
Approximately 6-10 minutes depending on workload startup time.

## Success Criteria
- Environment variables from container runtime are correctly detected
- Environment variables from manifest are properly handled (not in runtime detection)
- All workloads with environment variables are successfully instrumented
- Original application functionality is preserved after instrumentation
- No environment variable conflicts or overwrites occur

## Common Issues Detected
- **Variable Overwrites** - Instrumentation accidentally replacing application variables
- **Detection Failures** - Missing environment variables during runtime scan
- **Manifest Confusion** - Incorrectly detecting manifest variables as runtime variables
- **Preservation Issues** - Application variables not working after instrumentation

## Validation Points
- Correct detection source identification (runtime vs manifest)
- Proper null handling for non-runtime variables
- Successful instrumentation despite environment complexity
- Application functionality preservation post-instrumentation
