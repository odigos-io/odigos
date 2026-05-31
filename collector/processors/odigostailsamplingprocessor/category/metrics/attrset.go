package metrics

import (
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
	"go.opentelemetry.io/otel/attribute"
)

// CategoryMetricsAttributeSet returns an attribute set for the given category and dry run mode.
// used to record category level metrics.
func CategoryMetricsAttributeSet(category consts.SamplingCategory, dryRun bool) attribute.Set {
	categoryAttrs := []attribute.KeyValue{
		attribute.String(odigosattributes.SamplingCategory, string(category)),
	}
	if dryRun {
		categoryAttrs = append(categoryAttrs, attribute.Bool(odigosattributes.SamplingDryRun, true))
	}
	return attribute.NewSet(categoryAttrs...)
}
