package odigossqlqueryprocessor

import (
	"github.com/DataDog/go-sqllexer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv137 "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// defaultDBMS is used when db.system / db.system.name is missing or not mapped.
const defaultDBMS sqllexer.DBMSType = ""

// dbmsBySystem maps known SQL db.system / db.system.name values to sqllexer dialects.
var dbmsBySystem = map[string]sqllexer.DBMSType{
	// db.system (semconv v1.26)
	semconv.DBSystemMSSQL.Value.AsString():        sqllexer.DBMSSQLServer,
	semconv.DBSystemMssqlcompact.Value.AsString(): sqllexer.DBMSSQLServer,
	semconv.DBSystemPostgreSQL.Value.AsString():   sqllexer.DBMSPostgres,
	semconv.DBSystemMySQL.Value.AsString():        sqllexer.DBMSMySQL,
	semconv.DBSystemMariaDB.Value.AsString():      sqllexer.DBMSMySQL,
	semconv.DBSystemOracle.Value.AsString():       sqllexer.DBMSOracle,

	// db.system.name (semconv v1.37+)
	semconv137.DBSystemNameMicrosoftSQLServer.Value.AsString(): sqllexer.DBMSSQLServer,
	semconv137.DBSystemNamePostgreSQL.Value.AsString():         sqllexer.DBMSPostgres,
	semconv137.DBSystemNameMySQL.Value.AsString():              sqllexer.DBMSMySQL,
	semconv137.DBSystemNameMariaDB.Value.AsString():            sqllexer.DBMSMySQL,
	semconv137.DBSystemNameOracleDB.Value.AsString():           sqllexer.DBMSOracle,

	// sqllexer aliases / values that appear in the wild
	string(sqllexer.DBMSSQLServerAlias1): sqllexer.DBMSSQLServer, // sql-server
	string(sqllexer.DBMSSQLServerAlias2): sqllexer.DBMSSQLServer, // sqlserver
	string(sqllexer.DBMSPostgresAlias1):  sqllexer.DBMSPostgres,  // postgres
	string(sqllexer.DBMSSnowflake):       sqllexer.DBMSSnowflake, // snowflake (no semconv enum yet)
}

// nonSQLSystems are db.system / db.system.name values that are known not to be SQL
// and should skip obfuscation and attribute enhancement.
var nonSQLSystems = map[string]struct{}{
	// db.system (semconv v1.26)
	semconv.DBSystemCassandra.Value.AsString():     {},
	semconv.DBSystemHBase.Value.AsString():         {},
	semconv.DBSystemMongoDB.Value.AsString():       {},
	semconv.DBSystemRedis.Value.AsString():         {},
	semconv.DBSystemCouchbase.Value.AsString():     {},
	semconv.DBSystemCouchDB.Value.AsString():       {},
	semconv.DBSystemCosmosDB.Value.AsString():      {},
	semconv.DBSystemDynamoDB.Value.AsString():      {},
	semconv.DBSystemNeo4j.Value.AsString():         {},
	semconv.DBSystemGeode.Value.AsString():         {},
	semconv.DBSystemElasticsearch.Value.AsString(): {},
	semconv.DBSystemMemcached.Value.AsString():     {},
	semconv.DBSystemOpensearch.Value.AsString():    {},

	// db.system.name (semconv v1.37+)
	semconv137.DBSystemNameCassandra.Value.AsString():     {},
	semconv137.DBSystemNameHBase.Value.AsString():         {},
	semconv137.DBSystemNameMongoDB.Value.AsString():       {},
	semconv137.DBSystemNameRedis.Value.AsString():         {},
	semconv137.DBSystemNameCouchbase.Value.AsString():     {},
	semconv137.DBSystemNameCouchDB.Value.AsString():       {},
	semconv137.DBSystemNameAzureCosmosDB.Value.AsString(): {},
	semconv137.DBSystemNameAWSDynamoDB.Value.AsString():   {},
	semconv137.DBSystemNameNeo4j.Value.AsString():         {},
	semconv137.DBSystemNameGeode.Value.AsString():         {},
	semconv137.DBSystemNameElasticsearch.Value.AsString(): {},
	semconv137.DBSystemNameMemcached.Value.AsString():     {},
	semconv137.DBSystemNameOpenSearch.Value.AsString():    {},
	semconv137.DBSystemNameInfluxDB.Value.AsString():      {},
}

func dbSystemValue(attrs pcommon.Map) (string, bool) {
	for _, key := range []string{string(semconv137.DBSystemNameKey), string(semconv.DBSystemKey)} {
		val, ok := attrs.Get(key)
		if !ok || val.Type() != pcommon.ValueTypeStr {
			continue
		}
		system := val.Str()
		if system != "" {
			return system, true
		}
	}
	return "", false
}

// resolveDBMS reads db.system.name / db.system once (span first, then resource)
// and returns the sqllexer dialect and whether processing should be skipped for
// a known non-SQL system.
func resolveDBMS(spanAttrs pcommon.Map) (dbms sqllexer.DBMSType, skip bool) {
	system, found := dbSystemValue(spanAttrs)
	if !found {
		return defaultDBMS, false
	}
	if _, nonSQL := nonSQLSystems[system]; nonSQL {
		return defaultDBMS, true
	}
	if mapped, ok := dbmsBySystem[system]; ok {
		return mapped, false
	}
	return defaultDBMS, false
}
