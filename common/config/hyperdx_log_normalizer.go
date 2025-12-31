package config

// HyperdxLogNormalizerProcessor contains the OTTL transform processor config for HyperDX optimization.
// This processor parses JSON from log bodies, extracts severity levels, and normalizes log attributes.
//
// Copied from: https://github.com/hyperdxio/hyperdx/blobmain/docker/otel-collector/config.yaml
// Commit hash at time of copy: ab50b12a6b2492a1322af68ab3c618585da7764e
var HyperdxLogNormalizerProcessor = GenericMap{
	"log_statements": []GenericMap{
		// JSON parsing: Extends log attributes with fields from structured log body content
		{
			"context":    "log",
			"error_mode": "ignore",
			"statements": []string{
				`set(log.cache, ExtractPatterns(log.body, "(?P<0>\\{\\s*\".*\\})")) where IsString(log.body)`,
				`merge_maps(log.attributes, ParseJSON(log.cache["0"]), "upsert") where IsString(log.cache["0"]) and IsMatch(log.cache["0"], "^\\s*\\{\\s*\"")`,
				`flatten(log.attributes) where IsString(log.cache["0"]) and IsMatch(log.cache["0"], "^\\s*\\{\\s*\"")`,
				`merge_maps(log.attributes, log.body, "upsert") where IsMap(log.body)`,
			},
		},
		// Severity inference: extract log level from first 256 chars of body
		{
			"context":    "log",
			"error_mode": "ignore",
			"conditions": []string{
				`severity_number == 0 and severity_text == ""`,
			},
			"statements": []string{
				`set(log.cache["substr"], log.body.string) where Len(log.body.string) < 256`,
				`set(log.cache["substr"], Substring(log.body.string, 0, 256)) where Len(log.body.string) >= 256`,
				`set(log.cache, ExtractPatterns(log.cache["substr"], "(?i)(?P<0>(alert|crit|emerg|fatal|error|err|warn|notice|debug|dbug|trace))"))`,
				`set(log.severity_number, SEVERITY_NUMBER_FATAL) where IsMatch(log.cache["0"], "(?i)(alert|crit|emerg|fatal)")`,
				`set(log.severity_text, "fatal") where log.severity_number == SEVERITY_NUMBER_FATAL`,
				`set(log.severity_number, SEVERITY_NUMBER_ERROR) where IsMatch(log.cache["0"], "(?i)(error|err)")`,
				`set(log.severity_text, "error") where log.severity_number == SEVERITY_NUMBER_ERROR`,
				`set(log.severity_number, SEVERITY_NUMBER_WARN) where IsMatch(log.cache["0"], "(?i)(warn|notice)")`,
				`set(log.severity_text, "warn") where log.severity_number == SEVERITY_NUMBER_WARN`,
				`set(log.severity_number, SEVERITY_NUMBER_DEBUG) where IsMatch(log.cache["0"], "(?i)(debug|dbug)")`,
				`set(log.severity_text, "debug") where log.severity_number == SEVERITY_NUMBER_DEBUG`,
				`set(log.severity_number, SEVERITY_NUMBER_TRACE) where IsMatch(log.cache["0"], "(?i)(trace)")`,
				`set(log.severity_text, "trace") where log.severity_number == SEVERITY_NUMBER_TRACE`,
				`set(log.severity_text, "info") where log.severity_number == 0`,
				`set(log.severity_number, SEVERITY_NUMBER_INFO) where log.severity_number == 0`,
			},
		},
		// Normalize severity_text case
		{
			"context":    "log",
			"error_mode": "ignore",
			"statements": []string{
				`set(log.severity_text, ConvertCase(log.severity_text, "lower"))`,
			},
		},
	},
}
