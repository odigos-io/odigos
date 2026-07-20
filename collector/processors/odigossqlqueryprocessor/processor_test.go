package odigossqlqueryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func newTestProcessor(t *testing.T, cfg *Config) *sqlQueryProcessor {
	t.Helper()
	return newSqlQueryProcessor(processortest.NewNopSettings(typ), cfg)
}

func generateTestTrace(attrs map[string]string) ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("db")
	for k, v := range attrs {
		span.Attributes().PutStr(k, v)
	}
	return td
}

func TestInferAttributes_FromDbQueryText(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users WHERE id = 1",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	op, ok := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.True(t, ok)
	require.Equal(t, "SELECT", op.Str())

	table, ok := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.True(t, ok)
	require.Equal(t, "users", table.Str())
	require.Equal(t, "SELECT users", span.Name())

	query, _ := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.Equal(t, "SELECT * FROM users WHERE id = 1", query.Str())
}

func TestInferAttributes_FromDbStatement(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		dbStatementKey: "INSERT INTO orders VALUES (1)",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	op, ok := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.True(t, ok)
	require.Equal(t, "INSERT", op.Str())

	table, ok := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.True(t, ok)
	require.Equal(t, "orders", table.Str())
	require.Equal(t, "INSERT orders", span.Name())
}

func TestInferAttributes_PrefersDbQueryTextOverDbStatement(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users",
		dbStatementKey:                 "DELETE FROM orders",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	op, _ := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.Equal(t, "SELECT", op.Str())
	table, _ := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.Equal(t, "users", table.Str())
	require.Equal(t, "SELECT users", span.Name())
}

func TestInferAttributes_DoesNotOverwriteExisting(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey):      "SELECT * FROM users",
		string(semconv.DBOperationNameKey):  "EXISTING_OP",
		string(semconv.DBCollectionNameKey): "existing_table",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	op, _ := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.Equal(t, "EXISTING_OP", op.Str())
	table, _ := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.Equal(t, "existing_table", table.Str())
	require.Equal(t, "db", span.Name())
}

func TestInferAttributes_FillsMissingOnly(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey):     "UPDATE users SET name = 'x'",
		string(semconv.DBOperationNameKey): "EXISTING_OP",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	op, _ := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.Equal(t, "EXISTING_OP", op.Str())
	table, ok := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.True(t, ok)
	require.Equal(t, "users", table.Str())
	require.Equal(t, "EXISTING_OP users", span.Name())
}

func TestInferAttributes_Disabled(t *testing.T) {
	proc := newTestProcessor(t, &Config{})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	_, hasOp := span.Attributes().Get(string(semconv.DBOperationNameKey))
	_, hasTable := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.False(t, hasOp)
	require.False(t, hasTable)
	require.Equal(t, "db", span.Name())
}

func TestInferAttributes_IgnoresNonStringQuery(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("db")
	span.Attributes().PutInt(string(semconv.DBQueryTextKey), 123)

	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)

	outSpan := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	_, hasOp := outSpan.Attributes().Get(string(semconv.DBOperationNameKey))
	require.False(t, hasOp)
	require.Equal(t, "db", outSpan.Name())
}

func TestInferAttributes_MultipleTables(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	_, hasCollection := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.False(t, hasCollection)

	op, ok := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.True(t, ok)
	require.Equal(t, "SELECT", op.Str())
	require.Equal(t, "SELECT", span.Name())
}

func TestInferAttributes_OperationOnlySpanName(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT 1",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	op, ok := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.True(t, ok)
	require.Equal(t, "SELECT", op.Str())
	require.Equal(t, "SELECT", span.Name())
}

func TestRedactLiterals_Only(t *testing.T) {
	proc := newTestProcessor(t, &Config{RedactLiterals: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users WHERE id = 1 AND name = 'alice'",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.True(t, ok)
	require.Equal(t, "SELECT * FROM users WHERE id = ? AND name = ?", query.Str())

	_, hasOp := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.False(t, hasOp)
	require.Equal(t, "db", span.Name())
}

func TestRedactLiterals_WithInferAttributes(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true, RedactLiterals: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users WHERE id = 1 AND name = 'alice'",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.True(t, ok)
	require.Equal(t, "SELECT * FROM users WHERE id = ? AND name = ?", query.Str())

	op, ok := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.True(t, ok)
	require.Equal(t, "SELECT", op.Str())

	table, ok := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.True(t, ok)
	require.Equal(t, "users", table.Str())
	require.Equal(t, "SELECT users", span.Name())
}

func TestRedactLiterals_WhenAttributesAlreadyPresent(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true, RedactLiterals: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey):      "SELECT * FROM users WHERE id = 1",
		string(semconv.DBOperationNameKey):  "EXISTING_OP",
		string(semconv.DBCollectionNameKey): "existing_table",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.True(t, ok)
	require.Equal(t, "SELECT * FROM users WHERE id = ?", query.Str())

	op, _ := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.Equal(t, "EXISTING_OP", op.Str())
	table, _ := span.Attributes().Get(string(semconv.DBCollectionNameKey))
	require.Equal(t, "existing_table", table.Str())
	require.Equal(t, "db", span.Name())
}

func TestRedactLiterals_DbStatement(t *testing.T) {
	proc := newTestProcessor(t, &Config{RedactLiterals: true})
	traces := generateTestTrace(map[string]string{
		dbStatementKey: "INSERT INTO orders VALUES (1, 'x')",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(dbStatementKey)
	require.True(t, ok)
	require.Equal(t, "INSERT INTO orders VALUES (?, ?)", query.Str())
}

func TestRedactLiterals_UsesDbSystemDialect(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true, RedactLiterals: true})
	// MySQL hash comments are stripped only when DBMSMySQL is selected.
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users WHERE id = 1 # secret",
		string(semconv.DBSystemKey):    semconv.DBSystemMySQL.Value.AsString(),
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.True(t, ok)
	require.Equal(t, "SELECT * FROM users WHERE id = ?", query.Str())
	require.Equal(t, "SELECT users", span.Name())
}

func TestRedactLiterals_UnsupportedDbSystemUsesDefault(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true, RedactLiterals: true})
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): "SELECT * FROM users WHERE id = 1 # secret",
		string(semconv.DBSystemKey):    semconv.DBSystemHive.Value.AsString(),
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.True(t, ok)
	// Default dialect does not treat '#' as a MySQL comment.
	require.Equal(t, "SELECT * FROM users WHERE id = ? # secret", query.Str())
}

func TestInferAttributes_KeepsSpanNameWhenAlreadyPresent(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("SELECT users")
	span.Attributes().PutStr(string(semconv.DBQueryTextKey), "SELECT * FROM users WHERE id = 1")

	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)

	outSpan := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	require.Equal(t, "SELECT users", outSpan.Name())
	op, ok := outSpan.Attributes().Get(string(semconv.DBOperationNameKey))
	require.True(t, ok)
	require.Equal(t, "SELECT", op.Str())
}

func TestInferAttributes_UpdatesSpanNameWhenMissingCollection(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true})
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("SELECT")
	span.Attributes().PutStr(string(semconv.DBQueryTextKey), "SELECT * FROM users WHERE id = 1")

	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)

	outSpan := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	require.Equal(t, "SELECT users", outSpan.Name())
}

func TestSkipNonSQL_MongoDB(t *testing.T) {
	proc := newTestProcessor(t, &Config{InferAttributes: true, RedactLiterals: true})
	originalQuery := `{"find": "users", "filter": {"id": 1}}`
	traces := generateTestTrace(map[string]string{
		string(semconv.DBQueryTextKey): originalQuery,
		string(semconv.DBSystemKey):    semconv.DBSystemMongoDB.Value.AsString(),
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	query, ok := span.Attributes().Get(string(semconv.DBQueryTextKey))
	require.True(t, ok)
	require.Equal(t, originalQuery, query.Str())
	_, hasOp := span.Attributes().Get(string(semconv.DBOperationNameKey))
	require.False(t, hasOp)
	require.Equal(t, "db", span.Name())
}
