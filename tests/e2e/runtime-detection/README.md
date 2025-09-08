# Runtime Detection Test Suite

## Purpose
This test suite verifies that Odigos correctly detects programming languages, their versions, and runtime characteristics across different environments and configurations. It tests the core runtime detection functionality that determines which workloads can be instrumented.

## What is Tested

### Languages Covered
- **Node.js/JavaScript** - JavaScript runtime detection and version parsing
- **Java** - JVM-based applications with different implementations
- **Python** - Python interpreter detection with various configurations
- **.NET** - .NET Core/Framework applications with different libc types
- **C++** - Unsupported language detection

### Test Workloads

#### Node.js Runtime Detection 4
- `nodejs-unsupported-version` - Detects v8.17.0 (below minimum support)
- `nodejs-very-old-version` - Detects language but no version info available
- `nodejs-dockerfile-env` - Detects v20.17.0 with environment variables from container runtime
- `nodejs-manifest-env` - Detects v20.17.0 with environment variables from manifest

#### Java Runtime Detection 6
- `java-supported-version` - Detects v17.0.12+7 (standard OpenJDK)
- `java-azul` - Detects Azul Zulu JRE (alternative JVM implementation)
- `java-supported-docker-env` - Detects v17.0.12+7 with container runtime env vars
- `java-supported-manifest-env` - Detects v17.0.12+7 with manifest env vars
- `java-latest-version` - Detects latest Java version (no specific version assertion)
- `java-old-version` - Detects older Java version (v11.0.27+6)
- `java-unique-exec` - Detects Java without "java" keyword in exec path (v21.0.7+6)

#### Python Runtime Detection 5
- `python-alpine` - Detects v3.10.15 on Alpine Linux with secure execution mode
- `python-other-agent` - Detects existing agent conflicts (New Relic)
- `python-not-supported` - Detects v3.6.15 (unsupported version)
- `python-gunicorn-server` - Detects v3.8.20 with Gunicorn WSGI server

#### .NET Runtime Detection 2
- `dotnet8-musl` - Detects .NET 8 with musl libc (Alpine Linux)
- `dotnet6-glibc` - Detects .NET 6 with glibc (standard Linux)

#### Unsupported Language Detection
- `cpp-http-server` - Detects C++ as unsupported language

## Runtime Detection Fields Tested

### Core Fields
- **`language`** - Programming language identification (javascript, java, python, dotnet, cplusplus)
- **`runtimeVersion`** - Version string parsing and validation

### Environment Detection Fields
- **`envFromContainerRuntime`** - Environment variables detected from container runtime
- **`envFromManifest`** - Environment variables from Kubernetes manifest (implicitly tested via null checks)

### .NET-Specific Fields
- **`libCType`** - C library type detection (musl vs glibc)

### Agent Conflict Detection
- **`otherAgent`** - Detection of existing monitoring agents (New Relic)

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Verify installation → Start trace DB
```

### 2. Application Deployment
```
Deploy 18 test workloads → Wait for pods to be ready
```

### 3. Runtime Detection
```
Instrument namespace → Trigger runtime detection → Verify detection results
```

### 4. Validation
```
Assert runtime detection → Verify specific field values → Summary report
```

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

