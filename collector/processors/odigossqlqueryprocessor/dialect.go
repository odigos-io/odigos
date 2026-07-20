package odigossqlqueryprocessor

import (
	"github.com/DataDog/go-sqllexer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv137 "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// defaultDBMS is used when db.system / db.system.name is missing or unsupported.
const defaultDBMS sqllexer.DBMSType = ""

var supportedDialects = []sqllexer.DBMSType{
	defaultDBMS,
	sqllexer.DBMSSQLServer,
	sqllexer.DBMSPostgres,
	sqllexer.DBMSMySQL,
	sqllexer.DBMSOracle,
	sqllexer.DBMSSnowflake,
}

// dbmsBySystem maps db.system and db.system.name values to sqllexer dialects.
// Unsupported systems intentionally omit entries so callers fall back to defaultDBMS.
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

// noSQLSystems are known non-SQL db.system / db.system.name values that should
// skip SQL obfuscation and attribute enhancement.
var noSQLSystems = map[string]struct{}{
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

type dialectEngines struct {
	normalizer *sqllexer.Normalizer
	obfuscator *sqllexer.Obfuscator
}

func newDialectEngines(cfg *Config) map[sqllexer.DBMSType]*dialectEngines {
	engines := make(map[sqllexer.DBMSType]*dialectEngines, len(supportedDialects))
	for _, dbms := range supportedDialects {
		eng := &dialectEngines{}
		if cfg.EnhanceAttributes {
			eng.normalizer = sqllexer.NewNormalizer(
				sqllexer.WithCollectCommands(true),
				sqllexer.WithCollectTables(true),
			)
		}
		if cfg.Obfuscate {
			eng.obfuscator = sqllexer.NewObfuscator()
		}
		engines[dbms] = eng
	}
	return engines
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

func isNoSQLSystem(system string) bool {
	_, ok := noSQLSystems[system]
	return ok
}

// shouldSkipNoSQL reports whether span/resource db.system identifies a NoSQL database.
// Span attributes take precedence over resource attributes.
func shouldSkipNoSQL(spanAttrs, resourceAttrs pcommon.Map) bool {
	if system, found := dbSystemValue(spanAttrs); found {
		return isNoSQLSystem(system)
	}
	if system, found := dbSystemValue(resourceAttrs); found {
		return isNoSQLSystem(system)
	}
	return false
}

// dbmsFromAttributes returns the dialect for db.system.name / db.system.
// found is true when either attribute is present (even if unsupported).
func dbmsFromAttributes(attrs pcommon.Map) (dbms sqllexer.DBMSType, found bool) {
	system, found := dbSystemValue(attrs)
	if !found {
		return defaultDBMS, false
	}
	if mapped, ok := dbmsBySystem[system]; ok {
		return mapped, true
	}
	// Attribute present but unsupported dialect → default.
	return defaultDBMS, true
}

func (p *sqlQueryProcessor) enginesFor(spanAttrs, resourceAttrs pcommon.Map) (*dialectEngines, sqllexer.DBMSType) {
	dbms, found := dbmsFromAttributes(spanAttrs)
	if !found {
		dbms, _ = dbmsFromAttributes(resourceAttrs)
	}
	eng, ok := p.engines[dbms]
	if !ok {
		return p.engines[defaultDBMS], defaultDBMS
	}
	return eng, dbms
}
