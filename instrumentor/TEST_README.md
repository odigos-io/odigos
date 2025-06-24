# Instrumentor Testing Documentation

This document describes the comprehensive testing structure for the Odigos Instrumentor component.

## Test Structure Overview

The instrumentor testing is organized into multiple layers to ensure thorough coverage while maintaining fast feedback loops during development:

### 1. Unit Tests
Fast, isolated tests that don't require external dependencies like etcd or Kubernetes API server.

**Location**: Various `*_test.go` files
- `internal/pod/pod_test.go` - Pod utility functions
- `internal/webhook_env_injector/webhook_env_injector_test.go` - Environment injection logic
- `controllers/instrumentationconfig/common_test.go` - Instrumentation configuration logic
- `controllers/manager_test.go` - Manager creation and setup
- `instrumentor_unit_test.go` - Core instrumentor functionality

**Coverage**: 
- Pod utilities: 100%
- Webhook environment injector: 73.9%
- Instrumentation config: 38.7%
- Manager functionality: Comprehensive

### 2. Integration Tests
Tests that verify component interactions but still use mocked Kubernetes clients where possible.

**Location**: `integration_test.go` (tagged with `//go:build integration`)

**Features tested**:
- Pod affinity and environment injection workflows
- Manager and webhook integration  
- Error handling scenarios
- Complex environment variable scenarios (ValueFrom)
- Performance and scalability

### 3. Full Integration Tests
Tests that require a real or simulated Kubernetes environment with etcd.

**Location**: `instrumentor_test.go` (requires etcd)

## Running Tests

### Quick Unit Tests (Recommended for Development)
```bash
# Run all unit tests (no etcd required)
make test-unit

# Run manager unit tests
make test-manager

# Run with verbose output
make test-verbose
```

### Coverage Reports
```bash
# Generate unit test coverage report
make test-coverage

# View coverage in browser
open coverage.html
```

### Integration Tests
```bash
# Run integration tests (requires etcd)
make test-integration

# Generate integration test coverage
make test-coverage-integration
```

### All Tests
```bash
# Run all types of tests
make test-all

# Original test command (requires etcd)
make test
```

## Test Framework

### Ginkgo/Gomega
We use the Ginkgo BDD testing framework with Gomega matchers:

```go
Describe("Component", func() {
    BeforeEach(func() {
        // Setup
    })
    
    It("should do something", func() {
        // Test implementation
        Expect(result).To(Equal(expected))
    })
})
```

### Test Utilities
Located in `internal/testutil/`, provides helpers for:
- Setting Odigos instrumentation labels
- Creating mock objects
- Common test assertions

## Coverage Areas

### ✅ Well Tested Components

#### Pod Utilities (`internal/pod/`)
- **Coverage**: 100%
- **Tests**: 6 test cases
- **Features**: 
  - Odiglet affinity addition
  - Existing affinity preservation
  - Duplicate prevention
  - Edge cases with different operators

#### Webhook Environment Injector (`internal/webhook_env_injector/`)
- **Coverage**: 73.9% 
- **Tests**: 26 test cases
- **Features**:
  - Environment variable injection for multiple languages
  - ValueFrom environment variable handling
  - Runtime state validation
  - OTEL signal exporter configuration
  - Loader vs manifest injection methods

#### Manager (`controllers/manager.go`)
- **Tests**: 32 test cases  
- **Features**:
  - Manager creation with various configurations
  - Leader election configuration
  - Webhook registration
  - Scheme initialization

#### Instrumentation Config (`controllers/instrumentationconfig/`)
- **Coverage**: 38.7%
- **Tests**: 17 test cases
- **Features**:
  - Workload instrumentation configuration
  - Multiple language support
  - Rule-based configuration
  - HTTP payload collection rules

### 🔄 Integration Testing

#### End-to-End Workflows
- Pod processing with affinity and environment injection
- Manager lifecycle with different telemetry settings
- Error handling for invalid configurations
- Secure execution mode limitations
- Performance testing with concurrent pod processing

### 🎯 Areas for Future Improvement

1. **Controller Testing**: Individual controller tests (requires mocking or test environments)
2. **Error Path Coverage**: More comprehensive error scenario testing
3. **Performance Testing**: Load testing and benchmark tests
4. **Webhook Testing**: Direct webhook endpoint testing

## Writing New Tests

### Unit Test Example
```go
var _ = Describe("NewFeature", func() {
    var (
        component *Component
        logger    logr.Logger
    )

    BeforeEach(func() {
        logger = logr.Discard()
        component = NewComponent(logger)
    })

    It("should handle normal case", func() {
        result := component.DoSomething("input")
        Expect(result).To(Equal("expected"))
    })

    It("should handle error case", func() {
        result, err := component.DoSomethingRisky("bad-input")
        Expect(err).To(HaveOccurred())
        Expect(result).To(BeNil())
    })
})
```

### Integration Test Example
```go
It("should integrate components correctly", func() {
    By("Setting up component A")
    componentA := setupComponentA()
    
    By("Configuring component B") 
    componentB := setupComponentB()
    
    By("Testing interaction")
    result := componentA.InteractWith(componentB)
    Expect(result).To(BeTrue())
})
```

## Test Organization

### File Naming Convention
- `*_test.go` - Unit tests that can run without external dependencies
- `*_integration_test.go` - Integration tests that may require mocking
- Files with `//go:build integration` tag - Full integration tests

### Test Suite Naming
- "Component Unit Test Suite" - For pure unit tests
- "Component Integration Tests" - For integration tests
- "Component Suite" - For tests requiring full environment

## Contributing Test Guidelines

1. **Prefer unit tests** over integration tests for faster feedback
2. **Use table-driven tests** for testing multiple scenarios
3. **Test error paths** as well as happy paths
4. **Use descriptive test names** that explain the scenario
5. **Group related tests** in Describe blocks
6. **Use BeforeEach/AfterEach** for common setup/teardown
7. **Mock external dependencies** in unit tests
8. **Test edge cases** and boundary conditions

## Troubleshooting Tests

### Common Issues

#### etcd Not Found
```
fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory
```
**Solution**: Use unit tests (`make test-unit`) or install kubebuilder tools

#### Import Conflicts
**Solution**: Check for conflicting imports or variable naming

#### Timeout Issues
**Solution**: Increase timeout values or check for deadlocks

### Debug Tips

1. **Use `go test -v`** for verbose output
2. **Add `fmt.Printf`** for debugging test state
3. **Use `GinkgoWriter.Printf`** in Ginkgo tests
4. **Check coverage reports** to identify untested code paths
5. **Run individual test suites** to isolate issues

## Makefile Targets Reference

| Target | Description | Requirements |
|--------|-------------|--------------|
| `make test-unit` | Fast unit tests | None |
| `make test-manager` | Manager unit tests | None |  
| `make test-verbose` | Verbose unit tests | None |
| `make test-coverage` | Unit test coverage | None |
| `make test-integration` | Integration tests | etcd |
| `make test` | All tests | etcd |
| `make test-all` | All test types | etcd for integration |
| `make clean-test` | Clean test artifacts | None |