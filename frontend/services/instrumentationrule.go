package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deriveTypeFromRule(rule *model.InstrumentationRule) model.InstrumentationRuleType {
	if rule.CodeAttributes != nil {
		if rule.CodeAttributes.Column != nil || rule.CodeAttributes.FilePath != nil || rule.CodeAttributes.Function != nil || rule.CodeAttributes.LineNumber != nil || rule.CodeAttributes.Namespace != nil || rule.CodeAttributes.Stacktrace != nil {
			return model.InstrumentationRuleTypeCodeAttributes
		}
	}

	if rule.HeadersCollection != nil {
		if rule.HeadersCollection.HeaderKeys != nil {
			return model.InstrumentationRuleTypeHeadersCollection
		}
	}

	if rule.PayloadCollection != nil {
		if rule.PayloadCollection.HTTPRequest != nil || rule.PayloadCollection.HTTPResponse != nil || rule.PayloadCollection.DbQuery != nil || rule.PayloadCollection.Messaging != nil {
			return model.InstrumentationRuleTypePayloadCollection
		}
	}

	if rule.CustomInstrumentations != nil {
		return model.InstrumentationRuleTypeCustomInstrumentation
	}

	if rule.NetworkMetrics != nil && *rule.NetworkMetrics {
		return model.InstrumentationRuleTypeNetworkMetrics
	}

	if rule.KernelTraces != nil {
		return model.InstrumentationRuleTypeKernelTraces
	}

	return model.InstrumentationRuleTypeUnknownType
}

// getKernelTracesInput maps the GraphQL areas to the CRD's presence-based ones. It returns an error when
// no area is enabled, since the CRD requires at least one.
func getKernelTracesInput(input model.InstrumentationRuleInput) (*instrumentationrules.KernelTraces, error) {
	in := input.KernelTraces
	if in == nil {
		return nil, nil
	}
	out := &instrumentationrules.KernelTraces{}
	if in.Filesystem != nil && (in.Filesystem.Enabled == nil || *in.Filesystem.Enabled) {
		out.Filesystem = &instrumentationrules.KernelFilesystemTraces{}
	}
	if in.Subprocesses != nil {
		out.Subprocesses = &instrumentationrules.KernelSubprocessTraces{RecordCommandArgs: cloneBoolPtr(in.Subprocesses.RecordCommandArgs)}
	}
	if in.CPUDelay != nil && (in.CPUDelay.Enabled == nil || *in.CPUDelay.Enabled) {
		out.CPUDelay = &instrumentationrules.KernelCPUDelayTraces{}
	}
	if *out == (instrumentationrules.KernelTraces{}) {
		return nil, errors.New("at least one kernel category must be enabled")
	}
	return out, nil
}

func convertKernelTraces(traces *instrumentationrules.KernelTraces) *model.KernelTraces {
	if traces == nil {
		return nil
	}
	enabled := true
	out := &model.KernelTraces{}
	if traces.Filesystem != nil {
		out.Filesystem = &model.KernelFilesystemTraces{Enabled: &enabled}
	}
	if traces.Subprocesses != nil {
		out.Subprocesses = &model.KernelSubprocessTraces{RecordCommandArgs: cloneBoolPtr(traces.Subprocesses.RecordCommandArgs)}
	}
	if traces.CPUDelay != nil {
		out.CPUDelay = &model.KernelCPUDelayTraces{Enabled: &enabled}
	}
	return out
}

// getNetworkMetricsInput maps the presence-based GraphQL boolean to the CRD config: `true` becomes an
// empty `NetworkMetricsConfig{}` (enabled), while nil/false clears it (disabled).
func getNetworkMetricsInput(input model.InstrumentationRuleInput) *instrumentationrules.NetworkMetricsConfig {
	if input.NetworkMetrics == nil || !*input.NetworkMetrics {
		return nil
	}
	return &instrumentationrules.NetworkMetricsConfig{}
}

// networkMetricsEnabledPtr reports enablement back to the UI: a non-nil config becomes `true`, a nil
// config becomes nil (field omitted) so it doesn't spuriously mark other rule types.
func networkMetricsEnabledPtr(cfg *instrumentationrules.NetworkMetricsConfig) *bool {
	if cfg == nil {
		return nil
	}
	enabled := true
	return &enabled
}

// GetInstrumentationRules fetches all instrumentation rules
func GetInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	instrumentationRules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting instrumentation rules: %w", err)
	}

	var gqlRules []*model.InstrumentationRule
	for i := range instrumentationRules.Items {
		r := &instrumentationRules.Items[i]
		annotations := r.GetAnnotations()
		profileName := annotations[k8sconsts.OdigosProfileAnnotation]
		mutable := profileName == ""

		rule := &model.InstrumentationRule{
			RuleID:                   r.Name,
			RuleName:                 &r.Spec.RuleName,
			Notes:                    &r.Spec.Notes,
			Disabled:                 &r.Spec.Disabled,
			Mutable:                  mutable,
			ProfileName:              profileName,
			ManagedBy:                managedByFromLabels(r.Labels),
			SourcesScopes:            convertSourcesScope(r.Spec.Scopes),
			InstrumentationLibraries: convertInstrumentationLibraries(r.Spec.InstrumentationLibraries),
			Conditions:               ConvertConditions(r.Status.Conditions),
			CodeAttributes:           (*model.CodeAttributes)(r.Spec.CodeAttributes),
			HeadersCollection:        convertHeadersCollection(r.Spec.HeadersCollection),
			PayloadCollection:        convertPayloadCollection(r.Spec.PayloadCollection),
			CustomInstrumentations:   convertCustomInstrumentations(r.Spec.CustomInstrumentations),
			NetworkMetrics:           networkMetricsEnabledPtr(r.Spec.NetworkMetrics),
			KernelTraces:             convertKernelTraces(r.Spec.KernelTraces),
		}
		rule.Type = deriveTypeFromRule(rule)

		gqlRules = append(gqlRules, rule)
	}
	return gqlRules, nil
}

func GetInstrumentationRule(ctx context.Context, id string) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	r, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, handleNotFoundError(err, id, "instrumentation rule")
	}

	annotations := r.GetAnnotations()
	profileName := annotations[k8sconsts.OdigosProfileAnnotation]
	mutable := profileName == ""

	rule := &model.InstrumentationRule{
		RuleID:                   r.Name,
		RuleName:                 &r.Spec.RuleName,
		Notes:                    &r.Spec.Notes,
		Disabled:                 &r.Spec.Disabled,
		Mutable:                  mutable,
		ProfileName:              profileName,
		ManagedBy:                managedByFromLabels(r.Labels),
		SourcesScopes:            convertSourcesScope(r.Spec.Scopes),
		InstrumentationLibraries: convertInstrumentationLibraries(r.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(r.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(r.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(r.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(r.Spec.CustomInstrumentations),
		NetworkMetrics:           networkMetricsEnabledPtr(r.Spec.NetworkMetrics),
		KernelTraces:             convertKernelTraces(r.Spec.KernelTraces),
	}
	rule.Type = deriveTypeFromRule(rule)

	return rule, nil
}

func getPayloadCollectionInput(input model.InstrumentationRuleInput) *instrumentationrules.PayloadCollection {
	if input.PayloadCollection == nil {
		return nil
	}

	payloadCollection := &instrumentationrules.PayloadCollection{}

	if in := input.PayloadCollection.HTTPRequest; in != nil {
		payloadCollection.HttpRequest = fromHTTPPayloadInput(in)
	}
	if in := input.PayloadCollection.HTTPResponse; in != nil {
		payloadCollection.HttpResponse = fromHTTPPayloadInput(in)
	}
	if in := input.PayloadCollection.DbQuery; in != nil {
		payloadCollection.DbQuery = fromDbQueryPayloadInput(in)
	}
	if in := input.PayloadCollection.Messaging; in != nil {
		payloadCollection.Messaging = fromMessagingPayloadInput(in)
	}

	return payloadCollection
}

func mergePayloadCollectionUpdate(existing *instrumentationrules.PayloadCollection, input *model.PayloadCollectionInput) *instrumentationrules.PayloadCollection {
	if input == nil {
		return nil
	}

	out := &instrumentationrules.PayloadCollection{}
	if input.HTTPRequest != nil {
		out.HttpRequest = mergeHTTPPayloadUpdate(existingHTTPPayload(existing, payloadKindHTTPRequest), input.HTTPRequest)
	}
	if input.HTTPResponse != nil {
		out.HttpResponse = mergeHTTPPayloadUpdate(existingHTTPPayload(existing, payloadKindHTTPResponse), input.HTTPResponse)
	}
	if input.DbQuery != nil {
		out.DbQuery = mergeDbQueryPayloadUpdate(existingDbQueryPayload(existing), input.DbQuery)
	}
	if input.Messaging != nil {
		out.Messaging = mergeMessagingPayloadUpdate(existingMessagingPayload(existing), input.Messaging)
	}
	return out
}

// fromHTTPPayloadInput copies the advanced HTTP payload options (mime types,
// max length, drop-partial) from the GraphQL input into the CRD config. The
// GraphQL layer uses `[]*string` / `*int`, the CRD uses `*[]string` / `*int64`.
func fromHTTPPayloadInput(in *model.HTTPPayloadCollectionInput) *instrumentationrules.HttpPayloadCollection {
	cfg := &instrumentationrules.HttpPayloadCollection{
		MaxPayloadLength:    intToInt64Ptr(in.MaxPayloadLength),
		DropPartialPayloads: in.DropPartialPayloads,
	}
	if in.MimeTypes != nil {
		mimeTypes := make([]string, 0, len(in.MimeTypes))
		for _, m := range in.MimeTypes {
			if m != nil {
				mimeTypes = append(mimeTypes, *m)
			}
		}
		cfg.MimeTypes = &mimeTypes
	}
	return cfg
}

func fromDbQueryPayloadInput(in *model.DbQueryPayloadCollectionInput) *instrumentationrules.DbQueryPayloadCollection {
	return &instrumentationrules.DbQueryPayloadCollection{
		MaxPayloadLength:    intToInt64Ptr(in.MaxPayloadLength),
		DropPartialPayloads: in.DropPartialPayloads,
	}
}

func fromMessagingPayloadInput(in *model.MessagingPayloadCollectionInput) *instrumentationrules.MessagingPayloadCollection {
	return &instrumentationrules.MessagingPayloadCollection{
		MaxPayloadLength:    intToInt64Ptr(in.MaxPayloadLength),
		DropPartialPayloads: in.DropPartialPayloads,
	}
}

func mergeHTTPPayloadUpdate(existing *instrumentationrules.HttpPayloadCollection, in *model.HTTPPayloadCollectionInput) *instrumentationrules.HttpPayloadCollection {
	cfg := fromHTTPPayloadInput(in)
	if existing == nil {
		return cfg
	}

	// Older UI payload editors send `{}` for an enabled section. Treat omitted
	// advanced fields as "unchanged" during update so limits/filters are not
	// silently erased when a rule is opened and saved through that path.
	if in.MimeTypes == nil {
		cfg.MimeTypes = cloneStringSlicePtr(existing.MimeTypes)
	}
	if in.MaxPayloadLength == nil {
		cfg.MaxPayloadLength = cloneInt64Ptr(existing.MaxPayloadLength)
	}
	if in.DropPartialPayloads == nil {
		cfg.DropPartialPayloads = cloneBoolPtr(existing.DropPartialPayloads)
	}
	return cfg
}

func mergeDbQueryPayloadUpdate(existing *instrumentationrules.DbQueryPayloadCollection, in *model.DbQueryPayloadCollectionInput) *instrumentationrules.DbQueryPayloadCollection {
	cfg := fromDbQueryPayloadInput(in)
	if existing == nil {
		return cfg
	}
	if in.MaxPayloadLength == nil {
		cfg.MaxPayloadLength = cloneInt64Ptr(existing.MaxPayloadLength)
	}
	if in.DropPartialPayloads == nil {
		cfg.DropPartialPayloads = cloneBoolPtr(existing.DropPartialPayloads)
	}
	return cfg
}

func mergeMessagingPayloadUpdate(existing *instrumentationrules.MessagingPayloadCollection, in *model.MessagingPayloadCollectionInput) *instrumentationrules.MessagingPayloadCollection {
	cfg := fromMessagingPayloadInput(in)
	if existing == nil {
		return cfg
	}
	if in.MaxPayloadLength == nil {
		cfg.MaxPayloadLength = cloneInt64Ptr(existing.MaxPayloadLength)
	}
	if in.DropPartialPayloads == nil {
		cfg.DropPartialPayloads = cloneBoolPtr(existing.DropPartialPayloads)
	}
	return cfg
}

type payloadKind string

const (
	payloadKindHTTPRequest  payloadKind = "httpRequest"
	payloadKindHTTPResponse payloadKind = "httpResponse"
)

func existingHTTPPayload(existing *instrumentationrules.PayloadCollection, kind payloadKind) *instrumentationrules.HttpPayloadCollection {
	if existing == nil {
		return nil
	}
	switch kind {
	case payloadKindHTTPRequest:
		return existing.HttpRequest
	case payloadKindHTTPResponse:
		return existing.HttpResponse
	default:
		return nil
	}
}

func existingDbQueryPayload(existing *instrumentationrules.PayloadCollection) *instrumentationrules.DbQueryPayloadCollection {
	if existing == nil {
		return nil
	}
	return existing.DbQuery
}

func existingMessagingPayload(existing *instrumentationrules.PayloadCollection) *instrumentationrules.MessagingPayloadCollection {
	if existing == nil {
		return nil
	}
	return existing.Messaging
}

func cloneStringSlicePtr(in *[]string) *[]string {
	if in == nil {
		return nil
	}
	out := append([]string(nil), (*in)...)
	return &out
}

func cloneInt64Ptr(in *int64) *int64 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func intToInt64Ptr(v *int) *int64 {
	if v == nil {
		return nil
	}
	i := int64(*v)
	return &i
}

func int64ToIntPtr(v *int64) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func getHeadersCollectionInput(input model.InstrumentationRuleInput) *instrumentationrules.HttpHeadersCollection {
	if input.HeadersCollection == nil {
		return nil
	}

	headersCollection := &instrumentationrules.HttpHeadersCollection{}

	if input.HeadersCollection.HeaderKeys != nil {
		headersCollection.HeaderKeys = make([]string, 0, len(input.HeadersCollection.HeaderKeys))
		for _, key := range input.HeadersCollection.HeaderKeys {
			headersCollection.HeaderKeys = append(headersCollection.HeaderKeys, *key)
		}
	}

	return headersCollection
}

func getCodeAttributesInput(input model.InstrumentationRuleInput) *instrumentationrules.CodeAttributes {
	if input.CodeAttributes == nil {
		return nil
	}

	codeAttributes := &instrumentationrules.CodeAttributes{}

	if input.CodeAttributes.Column != nil {
		codeAttributes.Column = input.CodeAttributes.Column
	}
	if input.CodeAttributes.FilePath != nil {
		codeAttributes.FilePath = input.CodeAttributes.FilePath
	}
	if input.CodeAttributes.Function != nil {
		codeAttributes.Function = input.CodeAttributes.Function
	}
	if input.CodeAttributes.LineNumber != nil {
		codeAttributes.LineNumber = input.CodeAttributes.LineNumber
	}
	if input.CodeAttributes.Namespace != nil {
		codeAttributes.Namespace = input.CodeAttributes.Namespace
	}
	if input.CodeAttributes.Stacktrace != nil {
		codeAttributes.Stacktrace = input.CodeAttributes.Stacktrace
	}

	return codeAttributes
}

func getCustomInstrumentationsInput(input model.InstrumentationRuleInput) (*instrumentationrules.CustomInstrumentationsWithMetadata, error) {
	if input.CustomInstrumentations == nil {
		return nil, nil
	}
	customInstrumentations := &instrumentationrules.CustomInstrumentationsWithMetadata{}
	// Iterate Java custom probes and verify input
	if input.CustomInstrumentations.Java != nil {
		customInstrumentations.Java = make([]instrumentationrules.JavaCustomProbeWithMetadata, 0, len(input.CustomInstrumentations.Java))
		for _, probe := range input.CustomInstrumentations.Java {
			apiProbe := instrumentationrules.JavaCustomProbeWithMetadata{}
			if probe.ClassName != nil {
				apiProbe.ClassName = *probe.ClassName
			} else {
				apiProbe.ClassName = ""
			}
			if probe.MethodName != nil {
				apiProbe.MethodName = *probe.MethodName
			} else {
				apiProbe.MethodName = ""
			}
			apiProbe.CustomProbeMetadata = customProbeMetadataFromInput(probe.Notes, probe.Category, probe.AddedAt)
			customInstrumentations.Java = append(customInstrumentations.Java, apiProbe)
		}
	}

	if input.CustomInstrumentations.Golang != nil {
		customInstrumentations.Golang = make([]instrumentationrules.GolangCustomProbeWithMetadata, 0, len(input.CustomInstrumentations.Golang))
		for _, probe := range input.CustomInstrumentations.Golang {
			apiProbe := instrumentationrules.GolangCustomProbeWithMetadata{}
			if probe.PackageName != nil {
				apiProbe.PackageName = *probe.PackageName
			} else {
				apiProbe.PackageName = ""
			}
			if probe.FunctionName != nil {
				apiProbe.FunctionName = *probe.FunctionName
			} else {
				apiProbe.FunctionName = ""
			}
			if probe.ReceiverName != nil {
				apiProbe.ReceiverName = *probe.ReceiverName
			} else {
				apiProbe.ReceiverName = ""
			}
			if probe.ReceiverMethodName != nil {
				apiProbe.ReceiverMethodName = *probe.ReceiverMethodName
			} else {
				apiProbe.ReceiverMethodName = ""
			}
			apiProbe.CustomProbeMetadata = customProbeMetadataFromInput(probe.Notes, probe.Category, probe.AddedAt)
			customInstrumentations.Golang = append(customInstrumentations.Golang, apiProbe)
		}
	}

	if input.CustomInstrumentations.Php != nil {
		customInstrumentations.Php = make([]instrumentationrules.PhpCustomProbeWithMetadata, 0, len(input.CustomInstrumentations.Php))
		for _, probe := range input.CustomInstrumentations.Php {
			apiProbe := instrumentationrules.PhpCustomProbeWithMetadata{}
			if probe.ClassName != nil {
				apiProbe.ClassName = *probe.ClassName
			}
			if probe.FunctionName != nil {
				apiProbe.FunctionName = *probe.FunctionName
			}
			apiProbe.CustomProbeMetadata = customProbeMetadataFromInput(probe.Notes, probe.Category, probe.AddedAt)
			customInstrumentations.Php = append(customInstrumentations.Php, apiProbe)
		}
	}

	// Remove duplicate Golang probes
	uniqueGolangProbes := make([]instrumentationrules.GolangCustomProbeWithMetadata, 0, len(customInstrumentations.Golang))
	uniqGoProbes := make(map[instrumentationrules.GolangCustomProbeWithMetadata]struct{})
	for _, probe := range customInstrumentations.Golang {
		uniqGoProbes[probe] = struct{}{}
	}
	for probe := range uniqGoProbes {
		uniqueGolangProbes = append(uniqueGolangProbes, probe)
	}
	customInstrumentations.Golang = uniqueGolangProbes

	// Remove duplicate Java probes
	uniqueJavaProbes := make([]instrumentationrules.JavaCustomProbeWithMetadata, 0, len(customInstrumentations.Java))
	javaSeen := make(map[instrumentationrules.JavaCustomProbeWithMetadata]struct{})
	for _, probe := range customInstrumentations.Java {
		javaSeen[probe] = struct{}{}
	}
	for probe := range javaSeen {
		uniqueJavaProbes = append(uniqueJavaProbes, probe)
	}
	customInstrumentations.Java = uniqueJavaProbes

	// Remove duplicate PHP probes
	uniquePhpProbes := make([]instrumentationrules.PhpCustomProbeWithMetadata, 0, len(customInstrumentations.Php))
	phpSeen := make(map[instrumentationrules.PhpCustomProbeWithMetadata]struct{})
	for _, probe := range customInstrumentations.Php {
		phpSeen[probe] = struct{}{}
	}
	for probe := range phpSeen {
		uniquePhpProbes = append(uniquePhpProbes, probe)
	}
	customInstrumentations.Php = uniquePhpProbes

	if input.CustomInstrumentations.Cpp != nil {
		customInstrumentations.Cpp = make([]instrumentationrules.CppCustomProbeWithMetadata, 0, len(input.CustomInstrumentations.Cpp))
		for _, probe := range input.CustomInstrumentations.Cpp {
			apiProbe := instrumentationrules.CppCustomProbeWithMetadata{}
			if probe.Signature != nil {
				apiProbe.Signature = *probe.Signature
			}
			apiProbe.CustomProbeMetadata = customProbeMetadataFromInput(probe.Notes, probe.Category, probe.AddedAt)
			customInstrumentations.Cpp = append(customInstrumentations.Cpp, apiProbe)
		}
	}

	if input.CustomInstrumentations.Python != nil {
		customInstrumentations.Python = make([]instrumentationrules.PythonCustomProbeWithMetadata, 0, len(input.CustomInstrumentations.Python))
		pythonSeen := make(map[string]struct{})
		for _, probe := range input.CustomInstrumentations.Python {
			if probe == nil {
				continue
			}
			apiProbe := instrumentationrules.PythonCustomProbeWithMetadata{}
			if probe.Module != nil {
				apiProbe.Module = *probe.Module
			}
			if probe.Qualname != nil {
				apiProbe.Qualname = *probe.Qualname
			}
			for _, c := range probe.Captures {
				if c == nil {
					continue
				}
				capture := instrumentationrules.PythonCapture{}
				if c.Path != nil {
					capture.Path = *c.Path
				}
				if c.Mode != nil {
					capture.Mode = instrumentationrules.PythonCaptureMode(*c.Mode)
				}
				apiProbe.Captures = append(apiProbe.Captures, capture)
			}
			apiProbe.CustomProbeMetadata = customProbeMetadataFromInput(probe.Notes, probe.Category, probe.AddedAt)
			key, err := json.Marshal(apiProbe)
			if err != nil {
				return nil, err
			}
			if _, dup := pythonSeen[string(key)]; dup {
				continue
			}
			pythonSeen[string(key)] = struct{}{}
			customInstrumentations.Python = append(customInstrumentations.Python, apiProbe)
		}
	}

	if err := customInstrumentations.Verify(); err != nil {
		return nil, err
	}
	return customInstrumentations, nil
}

func UpdateInstrumentationRule(ctx context.Context, id string, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	// Retrieve existing rule
	existingRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, handleNotFoundError(err, id, "instrumentation rule")
	}
	// Update the existing rule's specification
	existingRule.Spec.RuleName = *input.RuleName
	existingRule.Spec.Notes = *input.Notes
	existingRule.Spec.Disabled = *input.Disabled

	// Selectors: an omitted field keeps the stored value, so a partial update never
	// widens a scoped rule, while an explicit null clears it (null scopes means the
	// entire cluster, null libraries means all libraries).
	if input.SourcesScopes.IsSet() {
		existingRule.Spec.Scopes = convertSourcesScopeInput(input.SourcesScopes.Value())
	}

	if input.InstrumentationLibraries.IsSet() {
		existingRule.Spec.InstrumentationLibraries = convertInstrumentationLibrariesInput(input.InstrumentationLibraries.Value())
	}

	// Type payloads are preserved when nil: clients send null for the payloads that
	// don't apply to the rule's type, so nil can't mean "clear" here.
	if input.PayloadCollection != nil {
		existingRule.Spec.PayloadCollection = mergePayloadCollectionUpdate(existingRule.Spec.PayloadCollection, input.PayloadCollection)
	}

	if input.CodeAttributes != nil {
		existingRule.Spec.CodeAttributes = getCodeAttributesInput(input)
	}

	if input.HeadersCollection != nil {
		existingRule.Spec.HeadersCollection = getHeadersCollectionInput(input)
	}

	if input.CustomInstrumentations != nil {
		customInstrumentations, err := getCustomInstrumentationsInput(input)
		if err != nil {
			return nil, fmt.Errorf("invalid custom instrumentations: %w", err)
		}
		existingRule.Spec.CustomInstrumentations = customInstrumentations
	}

	if input.NetworkMetrics != nil {
		existingRule.Spec.NetworkMetrics = getNetworkMetricsInput(input)
	}

	if input.KernelTraces != nil {
		kernelTraces, err := getKernelTracesInput(input)
		if err != nil {
			return nil, fmt.Errorf("invalid kernel traces: %w", err)
		}
		existingRule.Spec.KernelTraces = kernelTraces
	}
	// Update rule in Kubernetes
	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Update(ctx, existingRule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
	}

	annotations := updatedRule.GetAnnotations()
	profileName := annotations[k8sconsts.OdigosProfileAnnotation]

	rule := model.InstrumentationRule{
		RuleID:                   updatedRule.Name,
		RuleName:                 &updatedRule.Spec.RuleName,
		Notes:                    &updatedRule.Spec.Notes,
		Disabled:                 &updatedRule.Spec.Disabled,
		Mutable:                  profileName == "",
		ProfileName:              profileName,
		ManagedBy:                managedByFromLabels(updatedRule.Labels),
		SourcesScopes:            convertSourcesScope(updatedRule.Spec.Scopes),
		InstrumentationLibraries: convertInstrumentationLibraries(updatedRule.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(updatedRule.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(updatedRule.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(updatedRule.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(updatedRule.Spec.CustomInstrumentations),
		NetworkMetrics:           networkMetricsEnabledPtr(updatedRule.Spec.NetworkMetrics),
		KernelTraces:             convertKernelTraces(updatedRule.Spec.KernelTraces),
	}
	rule.Type = deriveTypeFromRule(&rule)
	return &rule, nil
}

func DeleteInstrumentationRule(ctx context.Context, id string) (bool, error) {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		return false, handleNotFoundError(err, id, "instrumentation rule")
	}

	return true, nil
}

func CreateInstrumentationRule(ctx context.Context, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	ruleName := *input.RuleName
	notes := *input.Notes
	disabled := *input.Disabled

	sourcesScopes := convertSourcesScopeInput(input.SourcesScopes.Value())
	instrumentationLibraries := convertInstrumentationLibrariesInput(input.InstrumentationLibraries.Value())

	customInstrumentations, err := getCustomInstrumentationsInput(input)
	if err != nil {
		return nil, fmt.Errorf("invalid custom instrumentations: %w", err)
	}

	kernelTraces, err := getKernelTracesInput(input)
	if err != nil {
		return nil, fmt.Errorf("invalid kernel traces: %w", err)
	}

	// Define the new rule spec based on the input
	newRule := &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ui-instrumentation-rule-",
			Labels: map[string]string{
				k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue,
			},
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:                 ruleName,
			Notes:                    notes,
			Disabled:                 disabled,
			Scopes:                   sourcesScopes,
			InstrumentationLibraries: instrumentationLibraries,
			CodeAttributes:           getCodeAttributesInput(input),
			HeadersCollection:        getHeadersCollectionInput(input),
			PayloadCollection:        getPayloadCollectionInput(input),
			CustomInstrumentations:   customInstrumentations,
			NetworkMetrics:           getNetworkMetricsInput(input),
			KernelTraces:             kernelTraces,
		},
	}
	// Create the rule in Kubernetes
	createdRule, err := CreateResourceWithGenerateName(ctx, func() (*v1alpha1.InstrumentationRule, error) {
		return kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Create(ctx, newRule, metav1.CreateOptions{})
	})
	if err != nil {
		return nil, fmt.Errorf("error creating instrumentation rule: %w", err)
	}

	// Convert to GraphQL model and return
	rule := model.InstrumentationRule{
		RuleID:                   createdRule.Name,
		RuleName:                 &createdRule.Spec.RuleName,
		Notes:                    &createdRule.Spec.Notes,
		Disabled:                 &createdRule.Spec.Disabled,
		Mutable:                  true, // New rules are always mutable
		ProfileName:              "",   // New rules are not associated with a profile
		ManagedBy:                managedByFromLabels(createdRule.Labels),
		SourcesScopes:            convertSourcesScope(createdRule.Spec.Scopes),
		InstrumentationLibraries: convertInstrumentationLibraries(createdRule.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(createdRule.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(createdRule.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(createdRule.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(createdRule.Spec.CustomInstrumentations),
		NetworkMetrics:           networkMetricsEnabledPtr(createdRule.Spec.NetworkMetrics),
		KernelTraces:             convertKernelTraces(createdRule.Spec.KernelTraces),
	}
	rule.Type = deriveTypeFromRule(&rule)

	return &rule, nil
}

func handleNotFoundError(err error, id string, entity string) error {
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("%s with ID %s not found", entity, id)
	}
	return fmt.Errorf("error getting %s: %w", entity, err)
}

// convertSourcesScopeInput folds the row-per-criterion GraphQL shape into the
// single-object tri-list CRD shape. Each input row contributes to at most one
// of Sources/Namespaces (workload identity vs. namespace-only) plus optionally
// Languages. The CRD model has no equivalent for the legacy `containerName`
// field, so it is dropped on conversion.
func convertSourcesScopeInput(scopes []*model.InstrumentationRuleSourcesScopeInput) *k8sconsts.SourcesScopes {
	if len(scopes) == 0 {
		return nil
	}
	out := &k8sconsts.SourcesScopes{}
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		name := DerefString(scope.WorkloadName)
		kind := DerefK8sResourceKind(scope.WorkloadKind)
		namespace := DerefString(scope.WorkloadNamespace)
		language := DerefSamplingWorkloadLanguage(scope.WorkloadLanguage)

		switch {
		case name != "" || kind != "":
			out.Sources = append(out.Sources, k8sconsts.PodWorkload{
				Name:      name,
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKind(kind),
			})
		case namespace != "":
			out.Namespaces = append(out.Namespaces, namespace)
		}
		if language != "" {
			out.Languages = append(out.Languages, language)
		}
	}
	if len(out.Sources) == 0 && len(out.Namespaces) == 0 && len(out.Languages) == 0 {
		return nil
	}
	return out
}

// convertSourcesScope unfolds the single-object tri-list CRD shape back into the
// row-per-criterion GraphQL shape: one row per Source, one per Namespace, one
// per Language. This is the inverse of convertSourcesScopeInput for the common
// case of single-dimension rows.
// convertInstrumentationLibrariesInput returns nil (all libraries) for a nil list and
// a pointer to an empty list for an explicit empty one.
func convertInstrumentationLibrariesInput(libraries []*model.InstrumentationLibraryGlobalIDInput) *[]v1alpha1.InstrumentationLibraryGlobalId {
	if libraries == nil {
		return nil
	}
	converted := make([]v1alpha1.InstrumentationLibraryGlobalId, len(libraries))
	for i, lib := range libraries {
		converted[i] = v1alpha1.InstrumentationLibraryGlobalId{
			Name:     lib.Name,
			SpanKind: common.SpanKind(*lib.SpanKind),
			Language: common.ProgrammingLanguage(*lib.Language),
		}
	}
	return &converted
}

func convertSourcesScope(scopes *k8sconsts.SourcesScopes) []*model.InstrumentationRuleSourcesScope {
	if scopes == nil {
		return nil
	}
	var gqlSourcesScope []*model.InstrumentationRuleSourcesScope
	for _, src := range scopes.Sources {
		row := &model.InstrumentationRuleSourcesScope{
			WorkloadName:      StringPtrIfNotEmpty(src.Name),
			WorkloadKind:      K8sResourceKindPtrIfNotEmpty(string(src.Kind)),
			WorkloadNamespace: StringPtrIfNotEmpty(src.Namespace),
		}
		gqlSourcesScope = append(gqlSourcesScope, row)
	}
	for _, ns := range scopes.Namespaces {
		ns := ns
		gqlSourcesScope = append(gqlSourcesScope, &model.InstrumentationRuleSourcesScope{
			WorkloadNamespace: &ns,
		})
	}
	for _, lang := range scopes.Languages {
		l := model.SamplingWorkloadLanguage(lang)
		gqlSourcesScope = append(gqlSourcesScope, &model.InstrumentationRuleSourcesScope{
			WorkloadLanguage: &l,
		})
	}
	return gqlSourcesScope
}

// Converts InstrumentationLibraries to GraphQL-compatible format
func convertInstrumentationLibraries(libraries *[]v1alpha1.InstrumentationLibraryGlobalId) []*model.InstrumentationLibraryGlobalID {
	var gqlLibraries []*model.InstrumentationLibraryGlobalID
	if libraries != nil {
		for _, lib := range *libraries {
			spanKind := model.SpanKind(lib.SpanKind)
			language := model.ProgrammingLanguage(lib.Language)
			gqlLibraries = append(gqlLibraries, &model.InstrumentationLibraryGlobalID{
				Name:     lib.Name,
				SpanKind: &spanKind,
				Language: &language,
			})
		}
	}
	return gqlLibraries
}

func convertHeadersCollection(headers *instrumentationrules.HttpHeadersCollection) *model.HeadersCollection {
	if headers == nil {
		return nil
	}

	headerKeys := make([]*string, len(headers.HeaderKeys))
	for i := range headers.HeaderKeys {
		headerKeys[i] = &headers.HeaderKeys[i]
	}

	return &model.HeadersCollection{
		HeaderKeys: headerKeys,
	}
}

// Converts PayloadCollection to GraphQL-compatible format
func convertPayloadCollection(payload *instrumentationrules.PayloadCollection) *model.PayloadCollection {
	if payload == nil {
		return nil
	}

	return &model.PayloadCollection{
		HTTPRequest:  toHTTPPayload(payload.HttpRequest),
		HTTPResponse: toHTTPPayload(payload.HttpResponse),
		DbQuery:      toDbQueryPayload(payload.DbQuery),
		Messaging:    toMessagingPayload(payload.Messaging),
	}
}

// Helpers to map the stored payload configs back to the GraphQL model, carrying
// the advanced options (mime types / max length / drop-partial) so they survive
// a round-trip and render in the UI.
func toHTTPPayload(payload *instrumentationrules.HttpPayloadCollection) *model.HTTPPayloadCollection {
	if payload == nil {
		return nil
	}
	out := &model.HTTPPayloadCollection{
		MaxPayloadLength:    int64ToIntPtr(payload.MaxPayloadLength),
		DropPartialPayloads: payload.DropPartialPayloads,
	}
	if payload.MimeTypes != nil {
		mimeTypes := make([]*string, 0, len(*payload.MimeTypes))
		for i := range *payload.MimeTypes {
			v := (*payload.MimeTypes)[i]
			mimeTypes = append(mimeTypes, &v)
		}
		out.MimeTypes = mimeTypes
	}
	return out
}

func toDbQueryPayload(payload *instrumentationrules.DbQueryPayloadCollection) *model.DbQueryPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.DbQueryPayloadCollection{
		MaxPayloadLength:    int64ToIntPtr(payload.MaxPayloadLength),
		DropPartialPayloads: payload.DropPartialPayloads,
	}
}

func toMessagingPayload(payload *instrumentationrules.MessagingPayloadCollection) *model.MessagingPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.MessagingPayloadCollection{
		MaxPayloadLength:    int64ToIntPtr(payload.MaxPayloadLength),
		DropPartialPayloads: payload.DropPartialPayloads,
	}
}

// Converts CustomInstrumentations to GraphQL-compatible format
func convertCustomInstrumentations(customInstruAsInstruRule *instrumentationrules.CustomInstrumentationsWithMetadata) *model.CustomInstrumentations {
	if customInstruAsInstruRule == nil {
		return nil
	}
	customInstruAsGqlModel := &model.CustomInstrumentations{}
	if customInstruAsInstruRule.Golang != nil {
		for _, golangProbe := range customInstruAsInstruRule.Golang {
			notes, category, addedAt := customProbeMetadataModel(golangProbe.CustomProbeMetadata)
			customInstruAsGqlModel.Golang = append(customInstruAsGqlModel.Golang, &model.GolangCustomProbe{
				PackageName:        &golangProbe.PackageName,
				FunctionName:       &golangProbe.FunctionName,
				ReceiverName:       &golangProbe.ReceiverName,
				ReceiverMethodName: &golangProbe.ReceiverMethodName,
				Notes:              notes,
				Category:           category,
				AddedAt:            addedAt,
			})
		}
	}
	if customInstruAsInstruRule.Java != nil {
		for _, javaProbe := range customInstruAsInstruRule.Java {
			notes, category, addedAt := customProbeMetadataModel(javaProbe.CustomProbeMetadata)
			customInstruAsGqlModel.Java = append(customInstruAsGqlModel.Java, &model.JavaCustomProbe{
				ClassName:  &javaProbe.ClassName,
				MethodName: &javaProbe.MethodName,
				Notes:      notes,
				Category:   category,
				AddedAt:    addedAt,
			})
		}
	}
	if customInstruAsInstruRule.Php != nil {
		for _, phpProbe := range customInstruAsInstruRule.Php {
			notes, category, addedAt := customProbeMetadataModel(phpProbe.CustomProbeMetadata)
			customInstruAsGqlModel.Php = append(customInstruAsGqlModel.Php, &model.PhpCustomProbe{
				ClassName:    &phpProbe.ClassName,
				FunctionName: &phpProbe.FunctionName,
				Notes:        notes,
				Category:     category,
				AddedAt:      addedAt,
			})
		}
	}
	if customInstruAsInstruRule.Cpp != nil {
		for _, cppProbe := range customInstruAsInstruRule.Cpp {
			notes, category, addedAt := customProbeMetadataModel(cppProbe.CustomProbeMetadata)
			customInstruAsGqlModel.Cpp = append(customInstruAsGqlModel.Cpp, &model.CppCustomProbe{
				Signature: &cppProbe.Signature,
				Notes:     notes,
				Category:  category,
				AddedAt:   addedAt,
			})
		}
	}
	if customInstruAsInstruRule.Python != nil {
		for _, pythonProbe := range customInstruAsInstruRule.Python {
			notes, category, addedAt := customProbeMetadataModel(pythonProbe.CustomProbeMetadata)
			var captures []*model.PythonCapture
			for _, c := range pythonProbe.Captures {
				path, mode := c.Path, string(c.Mode)
				captures = append(captures, &model.PythonCapture{Path: &path, Mode: &mode})
			}
			customInstruAsGqlModel.Python = append(customInstruAsGqlModel.Python, &model.PythonCustomProbe{
				Module:   &pythonProbe.Module,
				Qualname: &pythonProbe.Qualname,
				Captures: captures,
				Notes:    notes,
				Category: category,
				AddedAt:  addedAt,
			})
		}
	}
	return customInstruAsGqlModel
}

func customProbeMetadataFromInput(notes *string, category *model.CustomProbeCategory, addedAt *string) instrumentationrules.CustomProbeMetadata {
	var meta instrumentationrules.CustomProbeMetadata
	if notes != nil {
		meta.Notes = *notes
	}
	if category != nil {
		meta.Category = instrumentationrules.CustomProbeCategory(*category)
	}
	if addedAt != nil {
		meta.AddedAt = *addedAt
	}
	return meta
}

func customProbeMetadataModel(meta instrumentationrules.CustomProbeMetadata) (*string, *model.CustomProbeCategory, *string) {
	var category *model.CustomProbeCategory
	if meta.Category != "" {
		value := model.CustomProbeCategory(meta.Category)
		category = &value
	}
	return StringPtrIfNotEmpty(meta.Notes), category, StringPtrIfNotEmpty(meta.AddedAt)
}
