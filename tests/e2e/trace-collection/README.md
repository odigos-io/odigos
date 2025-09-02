# Trace Collection Test Suite

## Purpose
This test suite verifies end-to-end trace collection functionality, ensuring that instrumented applications successfully generate, transmit, and store traces in the configured observability backend.

## What is Tested

### End-to-End Trace Flow
- **Traffic generation** to instrumented applications
- **Trace creation** by instrumentation agents
- **Trace transmission** through the observability pipeline
- **Trace storage** in the destination backend
- **Trace verification** using database queries

### Multi-Phase Testing
- **Phase 1** - Initial trace collection with default service names
- **Phase 2** - Trace collection after service name updates (planned)
- **Phase 3** - Trace collection after configuration changes (planned)

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Start trace DB → Deploy all workloads
```

### 2. Instrumentation Setup
```
Instrument namespace → Add destination → Verify pipeline → Wait for instrumentation
```

### 3. Traffic Generation
```
Wait for rollouts → Generate HTTP traffic → Send requests to all services
```

### 4. Trace Verification
```
Wait for trace processing → Query trace database → Verify trace collection
```

## Test Workloads
Uses all 24 workloads from runtime-detection test to ensure comprehensive trace collection coverage across all supported languages and versions.

### Instrumented Services
- **Node.js applications** - All supported Node.js versions
- **Java applications** - All supported Java versions and JVM variants
- **Python applications** - All supported Python versions and configurations
- **.NET applications** - All supported .NET versions (6, 8) with different libc types
- **Unsupported applications** - C++ (should not generate traces)

## Traffic Generation

### Traffic Pattern
- **HTTP requests** sent to each instrumented service
- **Multiple requests** per service to ensure trace capture
- **Coordinated generation** across all workloads simultaneously
- **Job-based execution** using Kubernetes Job for reliable completion

### Traffic Generation Process
1. **Deployment Readiness** - Wait for all deployments to be fully rolled out
2. **Job Creation** - Apply traffic generation Kubernetes Job
3. **Traffic Execution** - Job sends HTTP requests to all services
4. **Completion Wait** - Wait for job to complete successfully
5. **Cleanup** - Remove traffic generation job

## Trace Verification

### Verification Method
- **Database Queries** - Query the simple trace database for collected traces
- **Service Name Matching** - Verify traces from expected services are present
- **Trace Content Validation** - Ensure traces contain expected attributes
- **Coverage Verification** - Confirm traces from all instrumented services

### Query Execution
Uses `simple_trace_db_query_runner.sh` script to:
- Execute JMESPath queries against trace database
- Search for traces from specific service names
- Validate trace attributes and metadata
- Report query results for verification

### Expected Traces
The test verifies traces are collected from instrumented services including:
- `nodejs-minimum-version`, `nodejs-latest-version`
- `java-supported-version`, `java-latest-version`, `java-old-version`, `java-azul`, `java-unique-exec`
- `python-latest-version`, `python-alpine`, `python-min-version`, `python-gunicorn-server`
- `dotnet8-glibc`, `dotnet8-musl`, `dotnet6-glibc`, `dotnet6-musl`

## Files Structure
```
trace-collection/
├── README.md                          # This file
├── chainsaw-test.yaml                 # Main test definition
├── 01-generate-traffic.yaml           # Traffic generation Kubernetes Job
├── 01-wait-for-trace.yaml            # Phase 1 trace verification queries
├── 02-wait-for-trace.yaml            # Phase 2 trace verification (future)
└── 03-wait-for-trace.yaml            # Phase 3 trace verification (future)
```

## Trace Verification Queries

### Phase 1 Verification (`01-wait-for-trace.yaml`)
Searches for traces from all instrumented services using JMESPath queries:
```jmespath
length([?span.serviceName == 'nodejs-minimum-version']) > `0` ||
length([?span.serviceName == 'java-supported-version']) > `0` ||
length([?span.serviceName == 'python-latest-version' && span.spanAttributes."http.route" == 'insert-random/']) > `0`
```

### Service-Specific Validations
- **Node.js services** - Basic service name matching
- **Java services** - Service name and JVM-specific attributes
- **Python services** - Service name plus HTTP route attributes
- **.NET services** - Service name and runtime-specific metadata

## Test Phases

### Current Implementation (Phase 1)
1. **Instrumentation Wait** - Wait for all workloads to be instrumented
2. **Traffic Generation** - Send HTTP requests to all services
3. **Trace Collection** - Allow time for trace processing and storage
4. **Verification** - Query database to confirm trace collection

### Future Phases (Planned)
- **Phase 2** - Trace collection after service name updates
- **Phase 3** - Trace collection after configuration changes

## Dependencies
- Requires all workloads from runtime-detection test
- Depends on instrumentation-lifecycle for proper instrumentation
- Uses simple trace database as destination
- Requires traffic generation job execution capability
- Depends on trace database query functionality

## Duration
Approximately 10-15 minutes including:
- Workload deployment and instrumentation (8-10 minutes)
- Traffic generation (2-3 minutes)
- Trace processing and verification (2-3 minutes)

## Success Criteria
- All instrumented workloads are ready and healthy
- Traffic generation completes successfully for all services
- Traces are successfully collected from instrumented applications
- Database queries return expected traces for all instrumented services
- No trace collection failures or errors reported
- Trace content includes expected service names and attributes

## Failure Scenarios Detected
- **Instrumentation Failures** - Applications not properly instrumented
- **Traffic Generation Issues** - HTTP requests failing or not reaching services
- **Trace Transmission Problems** - Traces not reaching the destination
- **Pipeline Failures** - Observability pipeline not processing traces correctly
- **Storage Issues** - Traces not being stored in the database
- **Query Failures** - Database queries not finding expected traces

## Trace Quality Validations
- **Service Identification** - Correct service names in traces
- **Request Correlation** - Traces correspond to generated traffic
- **Attribute Completeness** - Expected trace attributes are present
- **Timing Accuracy** - Trace timestamps align with traffic generation
- **Coverage Completeness** - All instrumented services produce traces

## Integration Points
- **Pipeline Integration** - Verifies full observability pipeline functionality
- **Destination Integration** - Confirms traces reach configured destination
- **Agent Integration** - Validates instrumentation agent trace generation
- **Network Integration** - Ensures trace transmission across network boundaries
