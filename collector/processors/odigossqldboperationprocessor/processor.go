package odigossqldboperationprocessor

import (
	"context"
	"strings"

	"github.com/xwb1989/sqlparser"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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

	ValueUnknown string = "UNKNOWN"
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
				_, collectionNameExists := span.Attributes().Get(string(semconv.DBCollectionNameKey))
				if operationNameExists && collectionNameExists {
					continue
				}

				// Detect the `db.operation.name` from the query text
				operationName, tableName := DetectSQLOperationAndTableName(dbQueryText.AsString())

				// Used to build "{operation} {table}" span name
				spanName := ""
				// Only set the `db.operation.name` if the detected operation name is not "UNKNOWN"
				if operationName != ValueUnknown && !operationNameExists {
					span.Attributes().PutStr(string(semconv.DBOperationNameKey), operationName)
					spanName = operationName
				}

				if tableName != ValueUnknown && !collectionNameExists {
					span.Attributes().PutStr(string(semconv.DBCollectionNameKey), tableName)
					if spanName != "" {
						spanName = spanName + " " + tableName
					}
				}

				// If we have a new span name, use that (but only if the current span name is
				// our default "DB" auto-instrumentation name).
				if spanName != "" && span.Name() == "DB" {
					span.SetName(spanName)
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
func DetectSQLOperationAndTableName(query string) (string, string) {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return ValueUnknown, ValueUnknown
	}

	stmt, err := sqlparser.Parse(query)
	if err == nil {
		switch stmt := stmt.(type) {
		case *sqlparser.Select:
			return "SELECT", getTableName(stmt.From)
		case *sqlparser.Update:
			return "UPDATE", getTableName(stmt.TableExprs)
		case *sqlparser.Insert:
			return "INSERT", stmt.Table.Name.String()
		case *sqlparser.Delete:
			return "DELETE", getTableName(stmt.TableExprs)
		}
	}

	return detectBasedOnFirstWord(query), ValueUnknown
}

func detectBasedOnFirstWord(query string) string {
	firstWord := extractFirstWord(query)

	// Convert the first word to uppercase for comparison
	firstWord = strings.ToUpper(firstWord)

	switch firstWord {
	case OperationSelect, OperationInsert, OperationUpdate, OperationDelete, OperationCreate, OperationDrop, OperationAlter:
		return firstWord
	default:
		return ValueUnknown
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

// getTableName extracts the table name from a SQL node.
func getTableName(node sqlparser.SQLNode) string {
	switch tableExpr := node.(type) {
	case sqlparser.TableName:
		return tableExpr.Name.String()
	case sqlparser.TableExprs:
		for _, expr := range tableExpr {
			if tableName, ok := expr.(*sqlparser.AliasedTableExpr); ok {
				if name, ok := tableName.Expr.(sqlparser.TableName); ok {
					return name.Name.String()
				}
			}
		}
	}
	return ValueUnknown
}
