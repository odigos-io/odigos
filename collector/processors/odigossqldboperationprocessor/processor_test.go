package odigossqldboperationprocessor

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/ptracetest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

func generateTestTrace(dbQueryText string, dbOperationNameExists bool) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	if dbQueryText != "" {
		span.Attributes().PutStr(string(semconv.DBQueryTextKey), dbQueryText)
	}
	if dbOperationNameExists {
		span.Attributes().PutStr(string(semconv.DBOperationNameKey), "EXISTING_OPERATION")
	}
	return td
}

func TestDBOperationProcessor_NoDbQueryText(t *testing.T) {
	logger, _ := zap.NewDevelopment() // Enable logging in development mode
	processor := &DBOperationProcessor{
		logger: logger,
		config: &Config{},
	}

	// Generate trace with no db.query.text attribute
	logger.Info("Running test: NoDbQueryText - No db.query.text attribute in span")
	traces := generateTestTrace("", false)

	logger.Info("Generated traces", zap.Any("Traces", traces))
	processedTraces, err := processor.processTraces(context.Background(), traces)

	logger.Info("Processed traces", zap.Any("ProcessedTraces", processedTraces))
	require.NoError(t, err)

	// Ensure the processed traces match the original
	logger.Info("Checking trace comparison")
	require.NoError(t, ptracetest.CompareTraces(traces, processedTraces))

	logger.Info("Test passed: NoDbQueryText")
}

func TestDBOperationProcessor_ExistingDbOperationName(t *testing.T) {
	logger, _ := zap.NewDevelopment() // Enable logging in development mode
	processor := &DBOperationProcessor{
		logger: logger,
		config: &Config{},
	}

	// Generate trace with an existing db.operation.name attribute
	logger.Info("Running test: ExistingDbOperationName - db.operation.name already exists")
	traces := generateTestTrace("SELECT * FROM users", true)

	logger.Info("Generated traces", zap.String("Query", "SELECT * FROM users"), zap.Any("TracesCount ", traces.SpanCount()))
	processedTraces, err := processor.processTraces(context.Background(), traces)

	logger.Info("Processed traces", zap.Any("ProcessedTraces", processedTraces))
	require.NoError(t, err)

	// Ensure that db.operation.name was not overwritten
	span := processedTraces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	attrValue, exists := span.Attributes().Get(string(semconv.DBOperationNameKey))

	// Print the current attribute value
	logger.Info("Checking db.operation.name", zap.String("OperationName", attrValue.AsString()), zap.Bool("Exists", exists))
	require.True(t, exists)
	require.Equal(t, "EXISTING_OPERATION", attrValue.AsString())

	logger.Info("Test passed: ExistingDbOperationName", zap.String("Detected Operation", attrValue.AsString()))
}

func TestDBOperationProcessor_SetDbOperationName(t *testing.T) {
	logger, _ := zap.NewDevelopment() // Enable logging in development mode
	processor := &DBOperationProcessor{
		logger: logger,
		config: &Config{},
	}

	testCases := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"INSERT INTO users VALUES(1)", "INSERT"},
		{"UPDATE users SET name='John'", "UPDATE"},
		{"DELETE FROM users WHERE id=1", "DELETE"},
		{"CREATE TABLE users", "CREATE"},
		{"", "UNKNOWN"},
	}

	for _, tc := range testCases {
		logger.Info("Running test: SetDbOperationName", zap.String("Query", tc.query), zap.String("Expected Operation", tc.expected))
		traces := generateTestTrace(tc.query, false)
		processedTraces, err := processor.processTraces(context.Background(), traces)

		require.NoError(t, err)

		// Ensure that db.operation.name was set correctly
		span := processedTraces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
		attrValue, exists := span.Attributes().Get(string(semconv.DBOperationNameKey))
		// Check if the attribute exists before asserting its value
		if exists {
			require.Equal(t, tc.expected, attrValue.AsString())
			logger.Info("Test passed: SetDbOperationName", zap.String("Query", tc.query), zap.String("Detected Operation", attrValue.AsString()))
		} else {
			require.Equal(t, tc.expected, "UNKNOWN")
			logger.Info("Test passed: SetDbOperationName", zap.String("Query", tc.query), zap.String("Detected Operation", "UNKNOWN"))
		}
	}
}
