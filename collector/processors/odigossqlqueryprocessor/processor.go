package odigossqlqueryprocessor

import (
	"context"
	"fmt"
	"strings"

	"github.com/DataDog/go-sqllexer"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/collector"
)

const dbStatementKey = "db.statement"

type sqlQueryProcessor struct {
	logger     *zap.Logger
	config     *Config
	normalizer *sqllexer.Normalizer
	obfuscator *sqllexer.Obfuscator

	// provider is set in Start() when odigos_config_extension is present.
	provider collector.OdigosConfigExtension
}

func newSqlQueryProcessor(set processor.Settings, cfg *Config) *sqlQueryProcessor {
	return &sqlQueryProcessor{
		logger: set.Logger,
		config: cfg,
		// Always available: per-source config from the extension may enable either option.
		normalizer: sqllexer.NewNormalizer(
			sqllexer.WithCollectCommands(true),
			sqllexer.WithCollectTables(true),
		),
		obfuscator: sqllexer.NewObfuscator(),
	}
}

// Start resolves odigos_config_extension for per-source config lookups.
func (p *sqlQueryProcessor) Start(ctx context.Context, host component.Host) error {
	if p.config.OdigosConfigExtension == nil {
		p.logger.Warn("odigos_config_extension unset, using static infer_attributes / redact_literals")
		return nil
	}
	extID := p.config.OdigosConfigExtension
	ext, ok := host.GetExtensions()[*extID]
	if !ok {
		return fmt.Errorf("odigos config extension %q not found", extID.String())
	}
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %q is not an OdigosConfigExtension (got %T)", extID.String(), ext)
	}
	p.provider = odigosExt
	if !p.provider.WaitForCacheSync(ctx) {
		p.logger.Warn("odigos config extension cache sync did not complete; some spans may be missed on startup")
	}
	return nil
}

func (p *sqlQueryProcessor) Shutdown(context.Context) error {
	p.provider = nil
	return nil
}

func (p *sqlQueryProcessor) processTraces(_ context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		srcCfg, ok := p.resolveSourceConfig(rs.Resource())
		if !ok {
			continue
		}
		inferAttributes := srcCfg.InferDbAttributes != nil
		redactLiterals := srcCfg.DbQueryTemplatization != nil && srcCfg.DbQueryTemplatization.TemplatizeLiterals
		if !inferAttributes && !redactLiterals {
			continue
		}

		scopeSpans := rs.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				p.processSpan(spans.At(k), inferAttributes, redactLiterals)
			}
		}
	}
	return traces, nil
}

// resolveSourceConfig returns per-source collector config from the extension when attached,
// otherwise a config derived from the legacy static Config fields.
func (p *sqlQueryProcessor) resolveSourceConfig(resource pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	if p.provider == nil {
		cfg := &commonapi.ContainerCollectorConfig{}
		if p.config.InferAttributes {
			cfg.InferDbAttributes = &actions.InferDbAttributesConfig{}
		}
		if p.config.RedactLiterals {
			cfg.DbQueryTemplatization = &actions.DbQueryTemplatizationConfig{TemplatizeLiterals: true}
		}
		return cfg, true
	}
	return p.provider.GetFromResource(resource)
}

func (p *sqlQueryProcessor) processSpan(span ptrace.Span, inferAttributes, redactLiterals bool) {
	attrs := span.Attributes()

	opAttr, hasOperation := attrs.Get(string(semconv.DBOperationNameKey))
	collAttr, hasCollection := attrs.Get(string(semconv.DBCollectionNameKey))
	inferNeeded := inferAttributes && !(hasOperation && hasCollection)

	if !inferNeeded && !redactLiterals {
		return
	}

	dbms, skip := resolveDBMS(attrs)
	if skip {
		return
	}

	query, queryKey, ok := sqlQueryFromAttributes(attrs)
	if !ok {
		return
	}

	switch {
	case redactLiterals && inferNeeded:
		normalized, meta, err := p.obfuscateAndNormalize(query, dbms)
		if err != nil {
			// this can be ok, for example if the attribute is not sql syntax
			p.logger.Debug("failed to obfuscate and normalize SQL query", zap.Error(err))
			return
		}
		attrs.PutStr(queryKey, normalized)
		p.enhanceFromMetadata(span, opAttr, hasOperation, collAttr, hasCollection, meta)
	case redactLiterals:
		attrs.PutStr(queryKey, p.obfuscate(query, dbms))
	case inferNeeded:
		meta, err := p.normalize(query, dbms)
		if err != nil {
			p.logger.Debug("failed to normalize SQL query", zap.Error(err))
			return
		}
		p.enhanceFromMetadata(span, opAttr, hasOperation, collAttr, hasCollection, meta)
	}
}

func (p *sqlQueryProcessor) obfuscateAndNormalize(query string, dbms sqllexer.DBMSType) (string, *sqllexer.StatementMetadata, error) {
	if dbms == defaultDBMS {
		return sqllexer.ObfuscateAndNormalize(query, p.obfuscator, p.normalizer)
	}
	return sqllexer.ObfuscateAndNormalize(query, p.obfuscator, p.normalizer, sqllexer.WithDBMS(dbms))
}

func (p *sqlQueryProcessor) obfuscate(query string, dbms sqllexer.DBMSType) string {
	if dbms == defaultDBMS {
		return p.obfuscator.Obfuscate(query)
	}
	return p.obfuscator.Obfuscate(query, sqllexer.WithDBMS(dbms))
}

func (p *sqlQueryProcessor) normalize(query string, dbms sqllexer.DBMSType) (*sqllexer.StatementMetadata, error) {
	if dbms == defaultDBMS {
		_, meta, err := p.normalizer.Normalize(query)
		return meta, err
	}
	_, meta, err := p.normalizer.Normalize(query, sqllexer.WithDBMS(dbms))
	return meta, err
}

func (p *sqlQueryProcessor) enhanceFromMetadata(
	span ptrace.Span,
	opAttr pcommon.Value,
	hasOperation bool,
	collAttr pcommon.Value,
	hasCollection bool,
	meta *sqllexer.StatementMetadata,
) {
	if meta == nil {
		return
	}

	attrs := span.Attributes()
	ops := sqlOperations(meta.Commands)
	added := false

	operation := ""
	if hasOperation {
		operation = opAttr.Str()
	} else if len(ops) == 1 {
		operation = ops[0]
		attrs.PutStr(string(semconv.DBOperationNameKey), operation)
		added = true
	}

	collection := ""
	if hasCollection {
		collection = collAttr.Str()
	} else if len(meta.Tables) == 1 {
		collection = meta.Tables[0]
		attrs.PutStr(string(semconv.DBCollectionNameKey), collection)
		added = true
	}

	if !added || operation == "" {
		return
	}
	if spanNameAlreadyHas(span.Name(), operation, collection) {
		return
	}
	if collection != "" {
		span.SetName(operation + " " + collection)
		return
	}
	span.SetName(operation)
}

func spanNameAlreadyHas(name, operation, collection string) bool {
	if !strings.Contains(name, operation) {
		return false
	}
	if collection != "" && !strings.Contains(name, collection) {
		return false
	}
	return true
}

func sqlQueryFromAttributes(attrs pcommon.Map) (query string, key string, ok bool) {
	for _, attrKey := range []string{string(semconv.DBQueryTextKey), dbStatementKey} {
		val, found := attrs.Get(attrKey)
		if !found || val.Type() != pcommon.ValueTypeStr {
			continue
		}
		query = val.Str()
		if query != "" {
			return query, attrKey, true
		}
	}
	return "", "", false
}

// sqlOperations returns SQL commands suitable for db.operation.name,
// excluding JOIN which is a clause rather than a top-level operation.
func sqlOperations(commands []string) []string {
	ops := make([]string, 0, len(commands))
	for _, c := range commands {
		if strings.EqualFold(c, "JOIN") {
			continue
		}
		ops = append(ops, c)
	}
	return ops
}
