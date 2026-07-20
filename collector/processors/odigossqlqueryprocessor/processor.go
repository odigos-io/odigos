package odigossqlqueryprocessor

import (
	"context"
	"strings"

	"github.com/DataDog/go-sqllexer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

const dbStatementKey = "db.statement"

var sqlQueryAttributeKeys = [...]string{string(semconv.DBQueryTextKey), dbStatementKey}

type sqlQueryProcessor struct {
	logger     *zap.Logger
	config     *Config
	normalizer *sqllexer.Normalizer
	obfuscator *sqllexer.Obfuscator
}

func newSqlQueryProcessor(set processor.Settings, cfg *Config) *sqlQueryProcessor {
	p := &sqlQueryProcessor{
		logger: set.Logger,
		config: cfg,
	}
	if cfg.InferAttributes {
		p.normalizer = sqllexer.NewNormalizer(
			sqllexer.WithCollectCommands(true),
			sqllexer.WithCollectTables(true),
		)
	}
	if cfg.RedactLiterals {
		p.obfuscator = sqllexer.NewObfuscator()
	}
	return p
}

func (p *sqlQueryProcessor) processTraces(_ context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	if !p.config.InferAttributes && !p.config.RedactLiterals {
		return traces, nil
	}

	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		scopeSpans := resourceSpans.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				p.processSpan(spans.At(k))
			}
		}
	}
	return traces, nil
}

func (p *sqlQueryProcessor) processSpan(span ptrace.Span) {
	attrs := span.Attributes()

	opAttr, hasOperation := attrs.Get(string(semconv.DBOperationNameKey))
	collAttr, hasCollection := attrs.Get(string(semconv.DBCollectionNameKey))
	inferNeeded := p.config.InferAttributes && !(hasOperation && hasCollection)

	if !inferNeeded && !p.config.RedactLiterals {
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
	case p.config.RedactLiterals && inferNeeded:
		normalized, meta, err := p.obfuscateAndNormalize(query, dbms)
		if err != nil {
			// this can be ok, for example if the attribute is not sql syntax
			p.logger.Debug("failed to obfuscate and normalize SQL query", zap.Error(err))
			return
		}
		attrs.PutStr(queryKey, normalized)
		p.enhanceFromMetadata(span, opAttr, hasOperation, collAttr, hasCollection, meta)
	case p.config.RedactLiterals:
		attrs.PutStr(queryKey, p.obfuscate(query, dbms))
	case inferNeeded:
		meta, err := p.normalize(query, dbms)
		if err != nil {
			p.logger.Debug("failed to normalize SQL query", zap.Error(err))
			return
		}
		p.enhanceFromMetadata(span, opAttr, hasOperation, collAttr, hasCollection, meta)
	}

	if p.config.RedactLiterals {
		p.redactOtherQueryAttributes(attrs, queryKey, dbms)
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

func (p *sqlQueryProcessor) redactOtherQueryAttributes(attrs pcommon.Map, selectedKey string, dbms sqllexer.DBMSType) {
	for _, attrKey := range sqlQueryAttributeKeys {
		if attrKey == selectedKey {
			continue
		}
		val, found := attrs.Get(attrKey)
		if !found || val.Type() != pcommon.ValueTypeStr || val.Str() == "" {
			continue
		}
		attrs.PutStr(attrKey, p.obfuscate(val.Str(), dbms))
	}
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
	for _, attrKey := range sqlQueryAttributeKeys {
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
