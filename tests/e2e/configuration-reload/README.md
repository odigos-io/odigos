# Configuration Reload Test Suite

## Purpose
This test suite verifies that the Odigos collector correctly reloads its configuration when actions are created or modified, ensuring that observability pipeline changes are applied dynamically without service restarts.

## What is Tested

### Configuration Management
- **Action creation** - Adding new observability actions to the system
- **Configuration generation** - Automatic collector configuration updates
- **Configuration reload** - Hot-reloading collector configuration without restart
- **Reload verification** - Confirming configuration changes are applied

### Action Types
- **Cluster Info Action** - Adds cluster information to traces and metrics
- **Configuration Pipeline** - Integration between actions and collector config
- **Dynamic Updates** - Real-time configuration changes without downtime

## Test Flow

### 1. Setup Phase
```
Prepare destination → Install Odigos → Start trace DB → Add destination
```

### 2. Pipeline Establishment
```
Verify pipeline ready → Establish baseline configuration
```

### 3. Action Creation
```
Create cluster info action → Verify action created → Wait for config update
```

### 4. Configuration Reload Verification
```
Monitor collector logs → Verify "Config updated" message → Confirm reload success
```

## Test Components

### Action Definition
Creates a cluster info action that:
- **Adds cluster metadata** to all telemetry data
- **Identifies the cluster** as 'e2e-test-cluster'
- **Enriches traces** with cluster information
- **Triggers configuration update** in the collector

### Configuration Monitoring
- **Log Monitoring** - Watches collector logs for configuration update messages
- **Reload Detection** - Identifies successful configuration reload events
- **Timeout Handling** - Ensures test doesn't wait indefinitely for reload

## Key Validations

### Action Creation Verification
- **Action Existence** - Cluster info action is created successfully
- **Action Configuration** - Action contains correct cluster information
- **Action Status** - Action is in ready/active state

### Configuration Reload Verification
- **Update Detection** - Collector logs show "Config updated" message
- **Timing Verification** - Configuration update occurs within expected timeframe
- **No Errors** - Configuration reload completes without errors

## Expected Behavior

### Action Processing Flow
1. **Action Creation** - New cluster info action is applied to Kubernetes
2. **Configuration Generation** - Odigos generates updated collector configuration
3. **Configuration Delivery** - New configuration is delivered to collector
4. **Hot Reload** - Collector reloads configuration without restart
5. **Log Confirmation** - Collector logs "Config updated" message

### Configuration Impact
- **Cluster Information** - All future telemetry includes cluster metadata
- **Pipeline Enhancement** - Observability pipeline gains cluster context
- **No Service Disruption** - Applications continue running normally during reload
- **Immediate Effect** - Configuration changes take effect immediately

## Files Structure
```
configuration-reload/
├── README.md                          # This file
└── chainsaw-test.yaml                 # Main test definition (reuses existing action files)
```

## Reused Components
- **Action Definition** - Reuses `03-create-action.yaml` from original workload-lifecycle test
- **Action Assertions** - Reuses `03-assert-action-created.yaml` for verification
- **Configuration** - Leverages existing collector configuration management

## Configuration Reload Process

### Step 1: Baseline Configuration
```
Install Odigos → Set up pipeline → Establish working configuration
```

### Step 2: Action Creation
```
Apply cluster info action → Verify action created → Trigger config generation
```

### Step 3: Reload Monitoring
```
Watch collector logs → Wait for "Config updated" → Verify successful reload
```

### Step 4: Validation
```
Confirm no errors → Verify configuration active → Test completion
```

## Log Monitoring Details

### Target Log Message
The test specifically looks for the message:
```
"Config updated"
```

### Monitoring Process
1. **Log Streaming** - Continuously monitor collector deployment logs
2. **Pattern Matching** - Search for the specific "Config updated" message
3. **Timeout Management** - Wait up to 200 seconds for the message
4. **Success Detection** - Test passes when message is found

### Collector Component
- **Target** - `deployment.apps/odigos-gateway` in `odigos-test` namespace
- **Log Source** - Collector container logs
- **Message Pattern** - Exact string match for "Config updated"

## Dependencies
- Requires Odigos collector with hot-reload capability
- Depends on action processing and configuration generation
- Uses cluster info action functionality
- Requires log monitoring and pattern matching

## Duration
Approximately 5-8 minutes including:
- Setup and pipeline establishment (3-4 minutes)
- Action creation and processing (1-2 minutes)
- Configuration reload and verification (1-2 minutes)

## Success Criteria
- Cluster info action is created successfully
- Action assertions pass (action exists and is properly configured)
- Collector logs show "Config updated" message within timeout period
- Configuration reload completes without errors
- No service disruption during configuration update

## Failure Scenarios Detected
- **Action Creation Failures** - Problems creating or applying the action
- **Configuration Generation Issues** - Failures in generating updated collector config
- **Reload Failures** - Collector unable to reload configuration
- **Timeout Issues** - Configuration update taking longer than expected
- **Error Conditions** - Errors during configuration processing or reload

## Configuration Management Validation
- **Dynamic Updates** - Configuration changes without service restart
- **Hot Reload** - Live configuration updates without downtime
- **Error Handling** - Graceful handling of configuration issues
- **Logging** - Proper logging of configuration update events
- **Consistency** - Configuration changes applied consistently across pipeline

## Integration Points
- **Action Processing** - Integration with Odigos action system
- **Configuration Pipeline** - Connection to collector configuration management
- **Log Management** - Integration with logging and monitoring systems
- **Service Mesh** - Configuration updates in service mesh environments
