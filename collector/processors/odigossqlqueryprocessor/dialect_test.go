package odigossqlqueryprocessor

import (
	"testing"

	"github.com/DataDog/go-sqllexer"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv137 "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestDbmsFromAttributes(t *testing.T) {
	tests := []struct {
		name      string
		attrs     map[string]string
		wantDBMS  sqllexer.DBMSType
		wantFound bool
	}{
		{
			name:      "missing",
			attrs:     nil,
			wantDBMS:  defaultDBMS,
			wantFound: false,
		},
		{
			name:      "db.system postgresql",
			attrs:     map[string]string{string(semconv.DBSystemKey): semconv.DBSystemPostgreSQL.Value.AsString()},
			wantDBMS:  sqllexer.DBMSPostgres,
			wantFound: true,
		},
		{
			name:      "db.system.name microsoft.sql_server",
			attrs:     map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameMicrosoftSQLServer.Value.AsString()},
			wantDBMS:  sqllexer.DBMSSQLServer,
			wantFound: true,
		},
		{
			name: "db.system.name preferred over db.system",
			attrs: map[string]string{
				string(semconv137.DBSystemNameKey): semconv137.DBSystemNameMySQL.Value.AsString(),
				string(semconv.DBSystemKey):        semconv.DBSystemPostgreSQL.Value.AsString(),
			},
			wantDBMS:  sqllexer.DBMSMySQL,
			wantFound: true,
		},
		{
			name:      "unsupported sql-like system falls back to default",
			attrs:     map[string]string{string(semconv.DBSystemKey): semconv.DBSystemHive.Value.AsString()},
			wantDBMS:  defaultDBMS,
			wantFound: true,
		},
		{
			name:      "oracle.db",
			attrs:     map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameOracleDB.Value.AsString()},
			wantDBMS:  sqllexer.DBMSOracle,
			wantFound: true,
		},
		{
			name:      "snowflake",
			attrs:     map[string]string{string(semconv.DBSystemKey): string(sqllexer.DBMSSnowflake)},
			wantDBMS:  sqllexer.DBMSSnowflake,
			wantFound: true,
		},
		{
			name:      "postgres alias",
			attrs:     map[string]string{string(semconv.DBSystemKey): string(sqllexer.DBMSPostgresAlias1)},
			wantDBMS:  sqllexer.DBMSPostgres,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			for k, v := range tt.attrs {
				attrs.PutStr(k, v)
			}
			dbms, found := dbmsFromAttributes(attrs)
			require.Equal(t, tt.wantFound, found)
			require.Equal(t, tt.wantDBMS, dbms)
		})
	}
}

func TestShouldSkipNoSQL(t *testing.T) {
	tests := []struct {
		name          string
		spanAttrs     map[string]string
		resourceAttrs map[string]string
		wantSkip      bool
	}{
		{
			name:     "missing system",
			wantSkip: false,
		},
		{
			name:      "mongodb db.system",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString()},
			wantSkip:  true,
		},
		{
			name:      "redis db.system.name",
			spanAttrs: map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameRedis.Value.AsString()},
			wantSkip:  true,
		},
		{
			name:      "aws.dynamodb",
			spanAttrs: map[string]string{string(semconv137.DBSystemNameKey): semconv137.DBSystemNameAWSDynamoDB.Value.AsString()},
			wantSkip:  true,
		},
		{
			name:      "postgresql not skipped",
			spanAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemPostgreSQL.Value.AsString()},
			wantSkip:  false,
		},
		{
			name:          "resource mongodb",
			resourceAttrs: map[string]string{string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString()},
			wantSkip:      true,
		},
		{
			name: "span sql wins over resource nosql",
			spanAttrs: map[string]string{
				string(semconv.DBSystemKey): semconv.DBSystemPostgreSQL.Value.AsString(),
			},
			resourceAttrs: map[string]string{
				string(semconv.DBSystemKey): semconv.DBSystemMongoDB.Value.AsString(),
			},
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
			require.Equal(t, tt.wantSkip, shouldSkipNoSQL(spanAttrs, resourceAttrs))
		})
	}
}
