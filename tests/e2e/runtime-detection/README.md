# Runtime Detection Test Suite

## Purpose
This test suite verifies that Odigos correctly detects programming languages and their versions across different runtime environments and configurations.

## What is Tested

### Languages Covered
- **Node.js** - JavaScript runtime detection
- **Java** - JVM-based applications
- **Python** - Python interpreter detection
- **.NET** - .NET Core/Framework applications
- **C++** - Unsupported language detection

### Test Scenarios

#### Node.js Runtime Detection
- `nodejs-unsupported-version` - Detects v8.17.0 (below minimum support)
- `nodejs-very-old-version` - Detects language but no version info
- `nodejs-minimum-version` - Detects v14.0.0 (minimum supported)
- `nodejs-latest-version` - Detects latest Node.js version

#### Java Runtime Detection
- `java-supported-version` - Detects v17.0.12+7 (standard OpenJDK)
- `java-azul` - Detects Azul Zulu JRE (alternative JVM)
- `java-latest-version` - Detects latest Java version
- `java-old-version` - Detects older Java version (v11)
- `java-unique-exec` - Detects Java without "java" keyword in exec path

#### Python Runtime Detection
- `python-latest-version` - Detects latest Python version
- `python-alpine` - Detects v3.10.15 on Alpine Linux
- `python-min-version` - Detects v3.8.0 (minimum supported)
- `python-not-supported` - Detects v3.6.15 (unsupported version)
- `python-other-agent` - Detects existing agent (New Relic) conflicts
- `python-gunicorn-server` - Detects Python with Gunicorn WSGI server

#### .NET Runtime Detection
- `dotnet8-musl` - Detects .NET 8 with musl libc (Alpine)
- `dotnet6-musl` - Detects .NET 6 with musl libc
- `dotnet8-glibc` - Detects .NET 8 with glibc (standard Linux)
- `dotnet6-glibc` - Detects .NET 6 with glibc

#### Unsupported Language Detection
- `cpp-http-server` - Detects C++ as unsupported language

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Verify installation → Start trace DB
```

### 2. Application Deployment
```
Deploy all 24 test workloads → Wait for pods to be ready
```

### 3. Runtime Detection
```
Instrument namespace → Trigger runtime detection → Verify detection results
```

### 4. Validation
```
Assert runtime detection → Verify Python instances → Summary report
```

## Key Validations

### Runtime Detection Assertions
- **Language identification** - Correct language detected for each workload
- **Version detection** - Accurate version strings where available
- **Support classification** - Proper supported/unsupported determination
- **Environment detection** - Container vs manifest environment variables

### Special Validations
- **Python Instance Count** - Verifies exactly 5 Python instrumentation instances are created (excluding unsupported Python version)
- **Version Boundaries** - Tests minimum, maximum, and unsupported version handling
- **JRE Variants** - Ensures different JVM implementations are detected correctly
- **Library Conflicts** - Detects existing monitoring agents

## Expected Outcomes

### Supported Workloads (Should be detected and marked for instrumentation)
- All Node.js versions ≥ 14.0.0
- All Java versions ≥ 8
- All Python versions ≥ 3.8
- All .NET versions (6, 8)

### Unsupported Workloads (Should be detected but not instrumented)
- Node.js < 14.0.0
- Python < 3.8
- C++ applications

### Runtime Information Captured
- Language name and version
- Execution environment details
- Environment variables from container runtime
- Security execution mode (where applicable)
- Library type (musl vs glibc for .NET)

## Files Structure
```
runtime-detection/
├── README.md                          # This file
├── chainsaw-test.yaml                 # Main test definition
├── 01-install-test-apps.yaml          # All 24 workload definitions
└── 01-assert-runtime-detected.yaml    # Runtime detection assertions
```

## Dependencies
- Requires Odigos installation with runtime detection capabilities
- Uses common trace database for destination
- Relies on public ECR images for test workloads

## Duration
Approximately 8-12 minutes depending on image pull times and cluster resources.

## Success Criteria
- All 24 workloads deploy successfully
- Runtime detection completes for all workloads
- Language and version information is accurate
- Supported/unsupported classification is correct
- Gunicorn instance count matches expected value (6)
