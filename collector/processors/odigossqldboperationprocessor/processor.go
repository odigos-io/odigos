package odigossqldboperationprocessor

import (
	"context"
	"strings"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

type DBOperationProcessor struct {
	logger *zap.Logger
}

const (
	OperationSelect  string = "SELECT"
	OperationInsert  string = "INSERT"
	OperationUpdate  string = "UPDATE"
	OperationDelete  string = "DELETE"
	OperationCreate  string = "CREATE"
	OperationDrop    string = "DROP"
	OperationAlter   string = "ALTER"
	OperationUnknown string = "UNKNOWN"
)

func (sp *DBOperationProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	resources := td.ResourceSpans()
	// Iterate over resources
	for r := 0; r < resources.Len(); r++ {
		scoreSpan := resources.At(r).ScopeSpans()

		// Iterate over scopes
		for j := 0; j < scoreSpan.Len(); j++ {
			ils := scoreSpan.At(j)

			// Iterate over spans
			for k := 0; k < ils.Spans().Len(); k++ {
				span := ils.Spans().At(k)

				// Get the `db.query.text`` attribute, If no found, continue to the next span
				dbQueryText, found := span.Attributes().Get(string(semconv.DBQueryTextKey))
				if !found {
					continue
				}

				// Check if `db.operation.name` is already defined, If already defined, continue to the next span
				_, operationNameExists := span.Attributes().Get(string(semconv.DBOperationNameKey))
				if operationNameExists {
					continue
				}

				// Detect the `db.operation.name` from the query text
				operationName := DetectSQLOperationName(dbQueryText.AsString())

				// Only set the `db.operation.name` if the detected operation name is not "UNKNOWN"
				if operationName != OperationUnknown {
					span.Attributes().PutStr(string(semconv.DBOperationNameKey), operationName)
				}
			}
		}
	}
	return td, nil
}

// DetectSQLType is a simple heuristic to determine the SQL operation by checking if
// the first word of the query is a common keyword (e.g., SELECT, INSERT, UPDATE, DELETE, CREATE).
// It returns the corresponding operation name or "UNKNOWN" if no match is found,
// providing an efficient, lightweight solution for quick query classification.
func DetectSQLOperationName(query string) string {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return OperationUnknown
	}

	firstWord := extractFirstWord(query)

	// Convert the first word to uppercase for comparison
	firstWord = strings.ToUpper(firstWord)

	switch firstWord {
	case OperationSelect, OperationInsert, OperationUpdate, OperationDelete, OperationCreate, OperationDrop, OperationAlter:
		return firstWord
	default:
		return OperationUnknown
	}
}

// This function handles common cases like trimming whitespace, tabs, and newlines.
// We avoid using `strings.Fields` here to prevent unnecessary allocations (especially for large queries).
// Instead, we iterate through the string only until we find the first space or delimiter, making it more efficient.
func extractFirstWord(query string) string {
	for i := 0; i < len(query); i++ {
		if query[i] == ' ' || query[i] == '\t' || query[i] == '\n' {
			return query[:i]
		}
	}

	// return the entire query (single-word case)
	return query
}
