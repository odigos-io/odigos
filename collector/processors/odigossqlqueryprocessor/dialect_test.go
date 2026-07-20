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
		name          string
		spanAttrs     map[string]string
		resourceAttrs map[string]string
		wantDBMS      sqllexer.DBMSType
		wantSkip      bool
	}{
		{
			name:     "missing",
			wantDBMS: defaultDBMS,
			wantSkip: false,
		},
		{
			name:      "db.system postgresql",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemPostgreSQL.Value.AsString()},
			wantDBMS:  sqllexer.DBMSPostgres,
			wantSkip:  false,
		},
		{
			name:      "db.system.name microsoft.sql_server",
			spanAttrs: map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameMicrosoftSQLServer.Value.AsString()},
			wantDBMS:  sqllexer.DBMSSQLServer,
			wantSkip:  false,
		},
		{
			name: "db.system.name preferred over db.system",
			spanAttrs: map[string]string{
				string(semconv137.DBSystemNameKey): semconv137.DBSystemNameMySQL.Value.AsString(),
				string(semconv.DBSystemKey):        semconv.DBSystemPostgreSQL.Value.AsString(),
			},
			wantDBMS: sqllexer.DBMSMySQL,
			wantSkip: false,
		},
		{
			name:      "unsupported sql-like system falls back to default",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemHive.Value.AsString()},
			wantDBMS:  defaultDBMS,
			wantSkip:  false,
		},
		{
			name:      "oracle.db",
			spanAttrs: map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameOracleDB.Value.AsString()},
			wantDBMS:  sqllexer.DBMSOracle,
			wantSkip:  false,
		},
		{
			name:      "snowflake",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): string(sqllexer.DBMSSnowflake)},
			wantDBMS:  sqllexer.DBMSSnowflake,
			wantSkip:  false,
		},
		{
			name:      "postgres alias",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): string(sqllexer.DBMSPostgresAlias1)},
			wantDBMS:  sqllexer.DBMSPostgres,
			wantSkip:  false,
		},
		{
			name:      "mongodb skipped",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString()},
			wantDBMS:  defaultDBMS,
			wantSkip:  true,
		},
		{
			name:      "redis db.system.name skipped",
			spanAttrs: map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameRedis.Value.AsString()},
			wantDBMS:  defaultDBMS,
			wantSkip:  true,
		},
		{
			name:      "aws.dynamodb skipped",
			spanAttrs: map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameAWSDynamoDB.Value.AsString()},
			wantDBMS:  defaultDBMS,
			wantSkip:  true,
		},
		{
			name:          "resource mongodb skipped",
			resourceAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString()},
			wantDBMS:      defaultDBMS,
			wantSkip:      true,
		},
		{
			name: "span sql wins over resource non-sql",
			spanAttrs: map[string]string{
				string(semconv.DBSystemKey): semconv.DBSystemPostgreSQL.Value.AsString(),
			},
			resourceAttrs: map[string]string{
				string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString(),
			},
			wantDBMS: sqllexer.DBMSPostgres,
			wantSkip: false,
		},
		{
			name: "resource sql used when span missing",
			resourceAttrs: map[string]string{
				string(semconv.DBSystemKey): semconv.DBSystemMySQL.Value.AsString(),
			},
			wantDBMS: sqllexer.DBMSMySQL,
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spanAttrs := pcommon.NewMap()
			for k, v := range tt.spanAttrs {
				spanAttrs.PutStr(k, v)
			}
			resourceAttrs := pcommon.NewMap()
			for k, v := range tt.resourceAttrs {
				resourceAttrs.PutStr(k, v)
			}
			dbms, skip := resolveDBMS(spanAttrs, resourceAttrs)
			require.Equal(t, tt.wantSkip, skip)
			require.Equal(t, tt.wantDBMS, dbms)
		})
	}
}
