package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/mergeconfig"
	"github.com/odigos-io/odigos/distros/distro"
)

// givin instrumentation rules for a specific container in a source,
// return the payload collection config that should be used for the container
func CalculatePayloadCollectionConfig(distro *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *instrumentationrules.PayloadCollection {

	// only calculate payload collection config if the distro supports it
	if distro.Traces == nil || distro.Traces.PayloadCollection == nil || !distro.Traces.PayloadCollection.Supported {
		return nil
	}

	var payloadCollection *instrumentationrules.PayloadCollection
	for _, irl := range *irls {
		payloadCollection = mergePayloadCollectionConfigs(payloadCollection, irl.Spec.PayloadCollection)
	}

	return payloadCollection
}

// givin 2 payload collection configs, return the merged config.
// for each field, we either merge commutative values (e.g. mime types), or take the most restrictive value (e.g. smaller max payload length, or drop partial payloads if one of the configs is set to drop).
func mergePayloadCollectionConfigs(p1 *instrumentationrules.PayloadCollection, p2 *instrumentationrules.PayloadCollection) *instrumentationrules.PayloadCollection {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	if p2.HttpRequest != nil {
		p1.HttpRequest = mergeHttpPayloadCollectionRules(p1.HttpRequest, p2.HttpRequest)
	}
	if p2.HttpResponse != nil {
		p1.HttpResponse = mergeHttpPayloadCollectionRules(p1.HttpResponse, p2.HttpResponse)
	}
	if p2.DbQuery != nil {
		p1.DbQuery = mergeDbPayloadCollectionRules(p1.DbQuery, p2.DbQuery)
	}
	if p2.Messaging != nil {
		p1.Messaging = mergeMessagingPayloadCollectionRules(p1.Messaging, p2.Messaging)
	}
	return p1
}

func mergeHttpPayloadCollectionRules(p1 *instrumentationrules.HttpPayloadCollection, p2 *instrumentationrules.HttpPayloadCollection) *instrumentationrules.HttpPayloadCollection {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	return &instrumentationrules.HttpPayloadCollection{
		MimeTypes:           mergeconfig.MergeStringArrays(p1.MimeTypes, p2.MimeTypes),
		MaxPayloadLength:    mergeconfig.MergeOptionalIntChooseLower(p1.MaxPayloadLength, p2.MaxPayloadLength),
		DropPartialPayloads: mergeconfig.MergeOptionalBools(p1.DropPartialPayloads, p2.DropPartialPayloads),
	}
}

func mergeDbPayloadCollectionRules(p1 *instrumentationrules.DbQueryPayloadCollection, p2 *instrumentationrules.DbQueryPayloadCollection) *instrumentationrules.DbQueryPayloadCollection {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	return &instrumentationrules.DbQueryPayloadCollection{
		MaxPayloadLength:    mergeconfig.MergeOptionalIntChooseLower(p1.MaxPayloadLength, p2.MaxPayloadLength),
		DropPartialPayloads: mergeconfig.MergeOptionalBools(p1.DropPartialPayloads, p2.DropPartialPayloads),
		SanitizationPolicy:  mergeDbQuerySanitizationPolicy(p1.SanitizationPolicy, p2.SanitizationPolicy),
	}
}

func mergeMessagingPayloadCollectionRules(p1 *instrumentationrules.MessagingPayloadCollection, p2 *instrumentationrules.MessagingPayloadCollection) *instrumentationrules.MessagingPayloadCollection {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	return &instrumentationrules.MessagingPayloadCollection{
		MaxPayloadLength:    mergeconfig.MergeOptionalIntChooseLower(p1.MaxPayloadLength, p2.MaxPayloadLength),
		DropPartialPayloads: mergeconfig.MergeOptionalBools(p1.DropPartialPayloads, p2.DropPartialPayloads),
	}
}

func mergeDbQuerySanitizationPolicy(p1 *consts.DbQuerySanitizationPolicy, p2 *consts.DbQuerySanitizationPolicy) *consts.DbQuerySanitizationPolicy {
	switch {
	case p1 == nil && p2 == nil:
		return nil
	case p1 == nil:
		return p2
	case p2 == nil:
		return p1
	default:
		if consts.DbQuerySanitizationPolicyPriority(*p1) >= consts.DbQuerySanitizationPolicyPriority(*p2) {
			return p1
		} else {
			return p2
		}
	}
}
