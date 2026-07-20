package odigossqlqueryprocessor

import (
	"testing"

	"github.com/DataDog/go-sqllexer"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv137 "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestResolveDBMS(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]string
		wantDBMS sqllexer.DBMSType
		wantSkip bool
	}{
		{
			name:     "missing",
			wantDBMS: defaultDBMS,
			wantSkip: false,
		},
		{
			name:     "db.system postgresql",
			attrs:    map[string]string{string(semconv.DBSystemKey): semconv.DBSystemPostgreSQL.Value.AsString()},
			wantDBMS: sqllexer.DBMSPostgres,
			wantSkip: false,
		},
		{
			name:     "db.system.name microsoft.sql_server",
			attrs:    map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameMicrosoftSQLServer.Value.AsString()},
			wantDBMS: sqllexer.DBMSSQLServer,
			wantSkip: false,
		},
		{
			name: "db.system.name preferred over db.system",
			attrs: map[string]string{
				string(semconv137.DBSystemNameKey): semconv137.DBSystemNameMySQL.Value.AsString(),
				string(semconv.DBSystemKey):        semconv.DBSystemPostgreSQL.Value.AsString(),
			},
			wantDBMS: sqllexer.DBMSMySQL,
			wantSkip: false,
		},
		{
			name:     "unsupported sql-like system falls back to default",
			attrs:    map[string]string{string(semconv.DBSystemKey): semconv.DBSystemHive.Value.AsString()},
			wantDBMS: defaultDBMS,
			wantSkip: false,
		},
		{
			name:     "oracle.db",
			attrs:    map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameOracleDB.Value.AsString()},
			wantDBMS: sqllexer.DBMSOracle,
			wantSkip: false,
		},
		{
			name:     "snowflake",
			attrs:    map[string]string{string(semconv.DBSystemKey): string(sqllexer.DBMSSnowflake)},
			wantDBMS: sqllexer.DBMSSnowflake,
			wantSkip: false,
		},
		{
			name:     "postgres alias",
			attrs:    map[string]string{string(semconv.DBSystemKey): string(sqllexer.DBMSPostgresAlias1)},
			wantDBMS: sqllexer.DBMSPostgres,
			wantSkip: false,
		},
		{
			name:     "mongodb skipped",
			attrs:    map[string]string{string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString()},
			wantDBMS: defaultDBMS,
			wantSkip: true,
		},
		{
			name:     "redis db.system.name skipped",
			attrs:    map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameRedis.Value.AsString()},
			wantDBMS: defaultDBMS,
			wantSkip: true,
		},
		{
			name:     "aws.dynamodb skipped",
			attrs:    map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameAWSDynamoDB.Value.AsString()},
			wantDBMS: defaultDBMS,
			wantSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			for k, v := range tt.attrs {
				attrs.PutStr(k, v)
			}
			dbms, skip := resolveDBMS(attrs)
			require.Equal(t, tt.wantSkip, skip)
			require.Equal(t, tt.wantDBMS, dbms)
		})
	}
}
