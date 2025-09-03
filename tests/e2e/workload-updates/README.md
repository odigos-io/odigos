# Workload Updates Test Suite

## Purpose
This test suite verifies that Odigos correctly handles workload manifest updates and re-applies instrumentation when workloads are modified, ensuring instrumentation remains functional through application lifecycle changes.

## What is Tested

### Workload Update Scenarios
- **Manifest template updates** - Changes to workload specifications
- **Service name updates** - Modified service names for trace differentiation
- **Re-instrumentation** - Instrumentation reapplied after workload changes
- **Rollout coordination** - Proper deployment rollout management during updates

### Update Types
- **Template specification changes** - Workload manifest template modifications
- **Service name changes** - Updating reported service names for tracing
- **Deployment rollouts** - Coordinated pod restarts for changes to take effect

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Start trace DB → Deploy initial workloads
```

### 2. Initial Instrumentation
```
Instrument namespace → Add destination → Verify pipeline → Confirm instrumentation
```

### 3. Workload Updates
```
Update manifest templates → Update service names → Trigger rollouts
```

### 4. Re-instrumentation Verification
```
Wait for rollout completion → Verify re-instrumentation → Validate functionality
```

## Test Phases

### Phase 1: Initial State
- Deploy workloads using instrumentation-lifecycle test apps
- Apply instrumentation to the namespace
- Verify initial instrumentation is working correctly
- Establish baseline functionality

### Phase 2: Manifest Template Updates
- Apply changes to workload manifest templates
- Verify that template updates are processed correctly
- Confirm workload update assertions pass
- Ensure no instrumentation is lost during template changes

### Phase 3: Service Name Updates
- Update service names for trace differentiation
- Apply new service name configuration
- Verify instrumentation config reflects new service names
- Confirm service name changes are properly tracked

### Phase 4: Coordinated Rollouts
- Trigger deployment rollouts for all affected workloads
- Wait for all rollouts to complete successfully
- Verify that instrumentation remains active after rollouts
- Confirm all workloads are healthy post-update

## Key Validations

### Template Update Verification
- **Update Processing** - Template changes are correctly applied
- **Instrumentation Preservation** - Instrumentation survives template updates
- **Configuration Sync** - InstrumentationConfig reflects template changes

### Service Name Update Verification
- **Name Changes** - Service names are updated in instrumentation config
- **Trace Differentiation** - New service names enable trace phase identification
- **Configuration Updates** - InstrumentationConfig shows updated service names

### Re-instrumentation Validation
- **Agent Status** - Instrumentation agents remain enabled after updates
- **Health Reporting** - All workloads report healthy instrumentation status
- **Rollout Success** - All deployment rollouts complete without errors
- **Functionality** - Applications continue working normally after updates

## Expected Behavior

### During Template Updates
1. **Template Processing** - New manifest templates are applied
2. **Instrumentation Retention** - Existing instrumentation is preserved
3. **Configuration Update** - InstrumentationConfig reflects changes
4. **No Service Disruption** - Applications continue running during updates

### During Service Name Updates
1. **Name Application** - New service names are applied to configuration
2. **Trace Preparation** - Service names prepared for multi-phase trace collection
3. **Config Synchronization** - All instrumentation configs updated consistently

### During Rollouts
1. **Coordinated Restart** - All affected workloads restart together
2. **Instrumentation Reapplication** - Instrumentation devices re-injected
3. **Health Recovery** - All workloads return to healthy state
4. **Functionality Verification** - Applications work correctly post-rollout
