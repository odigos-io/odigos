# Instrumentation Lifecycle Test Suite

## Purpose
This test suite verifies the complete instrumentation lifecycle, from device injection to instance creation and health reporting for supported runtime environments.

## What is Tested

### Core Instrumentation Process
- **Device injection** into supported workload containers
- **Instrumentation instance creation** with proper metadata
- **Health status reporting** from instrumentation agents
- **Workload rollout management** during instrumentation

### Test Workloads
- `nodejs-minimum-version` - Node.js v14.0.0 (minimum supported)
- `nodejs-latest-version` - Latest Node.js version
- `java-supported-version` - Java v17.0.12+7
- `python-latest-version` - Latest Python version

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Verify installation → Start trace DB
```

### 2. Application Deployment
```
Deploy test workloads → Wait for pods to be ready
```

### 3. Instrumentation Setup
```
Instrument namespace → Add destination → Verify pipeline ready
```

### 4. Instrumentation Lifecycle
```
Assert instrumentation applied → Verify workload state changes → Check instances created
```

### 5. Validation
```
Verify instrumentation instances → Confirm workload rollouts → Health checks
```

## Key Validations

### Instrumentation Configuration Assertions
Each workload should have an InstrumentationConfig with:
- **MarkedForInstrumentation** - Workload selected for instrumentation
- **RuntimeDetection** - Runtime successfully detected
- **AgentEnabled** - Instrumentation agent enabled in containers
- **WorkloadRollout** - Deployment rollout triggered successfully

### Workload State Changes
Verifies that instrumented deployments show:
- **Revision increment** - `deployment.kubernetes.io/revision` increases to '2'
- **Generation update** - `metadata.generation` increments to 2
- **ObservedGeneration sync** - `status.observedGeneration` matches generation
- **Replica availability** - All replicas remain available during rollout

### Instrumentation Instance Creation
- **Instance count** - Correct number of InstrumentationInstance objects created
- **Metadata linking** - Instances properly linked to their workloads
- **Health reporting** - Instances report successful instrumentation

## Expected Behavior

### For Supported Workloads
1. **Device Injection** - Instrumentation device added to container spec
2. **Environment Setup** - Required environment variables injected
3. **Rollout Trigger** - Deployment automatically rolled out with new spec
4. **Agent Startup** - Instrumentation agent starts and reports health
5. **Instance Creation** - InstrumentationInstance created with success status

### Workload State Tracking
- **Before Instrumentation**: revision='1', generation=1, observedGeneration=1
- **After Instrumentation**: revision='2', generation=2, observedGeneration=2
- **Rollout Status**: All replicas updated and available

## Files Structure
```
instrumentation-lifecycle/
├── README.md                          # This file
├── chainsaw-test.yaml                 # Main test definition
├── 01-install-test-apps.yaml          # Core workload definitions
├── 01-assert-instrumented.yaml        # Instrumentation status assertions
└── 01-assert-workloads.yaml          # Deployment state change assertions
```

## Test Phases

### Phase 1: Pre-Instrumentation
- Deploy workloads in clean state
- Verify baseline deployment status
- Confirm no instrumentation present

### Phase 2: Instrumentation Application
- Apply namespace instrumentation
- Add observability destination
- Wait for pipeline readiness

### Phase 3: Lifecycle Verification
- Assert instrumentation configs created
- Verify workload state changes
- Confirm instance creation

### Phase 4: Health Validation
- Check instrumentation instance health
- Verify workload rollout completion
- Validate agent status reporting

## Dependencies
- Requires runtime-detection capabilities (language detection)
- Uses common trace database destination
- Depends on instrumentation device availability
- Requires workload rollout management

## Duration
Approximately 8-12 minutes including workload rollout time.

## Success Criteria
- All 4 test workloads are successfully instrumented
- InstrumentationConfig shows all conditions as "True"
- Workload deployments show proper state changes (revision increment)
- InstrumentationInstance objects are created for each workload
- All workload rollouts complete successfully
- No instrumentation errors or failures reported

## Failure Scenarios Detected
- **Runtime not supported** - Workloads with unsupported runtimes are skipped
- **Device injection failure** - Problems adding instrumentation to containers
- **Rollout issues** - Deployments that fail to update properly
- **Agent startup failure** - Instrumentation agents that fail to initialize
- **Health reporting failure** - Agents that don't report successful startup
