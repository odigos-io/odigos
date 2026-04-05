package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/mergeconfig"
	"github.com/odigos-io/odigos/distros/distro"
)

// givin instrumentation rules for a specific container in a source,
// return the payload collection config that should be used for the container
func CalculatePayloadCollectionConfig(distro *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule, language common.ProgrammingLanguage) *instrumentationrules.PayloadCollection {

	// only calculate payload collection config if the distro supports it
	if distro.Traces == nil || distro.Traces.PayloadCollection == nil || !distro.Traces.PayloadCollection.Supported {
		return nil
	}

	var payloadCollection *instrumentationrules.PayloadCollection
	for _, irl := range *irls {
		if !payloadCollectionRuleAppliesToLanguage(&irl, language) {
			continue
		}
		payloadCollection = mergePayloadCollectionConfigs(payloadCollection, irl.Spec.PayloadCollection)
	}

	return payloadCollection
}

func payloadCollectionRuleAppliesToLanguage(irl *odigosv1.InstrumentationRule, language common.ProgrammingLanguage) bool {
	if irl.Spec.InstrumentationLibraries == nil {
		return true
	}
	for _, library := range *irl.Spec.InstrumentationLibraries {
		if library.Language == language {
			return true
		}
	}
	return false
}

// givin 2 payload collection configs, return the merged config.
// for each field, we either merge commutative values (e.g. mime types), or take the most restrictive value (e.g. smaller max payload length, or drop partial payloads if one of the configs is set to drop).
func mergePayloadCollectionConfigs(p1 *instrumentationrules.PayloadCollection, p2 *instrumentationrules.PayloadCollection) *instrumentationrules.PayloadCollection {
	if p1 == nil {
		if p2 == nil {
			return nil
		}
		return p2.DeepCopy()
	}
	if p2 == nil {
		return p1.DeepCopy()
	}
	merged := p1.DeepCopy()
	if p2.HttpRequest != nil {
		merged.HttpRequest = mergeHttpPayloadCollectionRules(merged.HttpRequest, p2.HttpRequest)
	}
	if p2.HttpResponse != nil {
		merged.HttpResponse = mergeHttpPayloadCollectionRules(merged.HttpResponse, p2.HttpResponse)
	}
	if p2.DbQuery != nil {
		merged.DbQuery = mergeDbPayloadCollectionRules(merged.DbQuery, p2.DbQuery)
	}
	if p2.Messaging != nil {
		merged.Messaging = mergeMessagingPayloadCollectionRules(merged.Messaging, p2.Messaging)
	}
	return merged
}

func mergeHttpPayloadCollectionRules(p1 *instrumentationrules.HttpPayloadCollection, p2 *instrumentationrules.HttpPayloadCollection) *instrumentationrules.HttpPayloadCollection {
	if p1 == nil {
		if p2 == nil {
			return nil
		}
		return p2.DeepCopy()
	}
	if p2 == nil {
		return p1.DeepCopy()
	}
	return &instrumentationrules.HttpPayloadCollection{
		MimeTypes:           mergeconfig.MergeStringArrays(p1.MimeTypes, p2.MimeTypes),
		MaxPayloadLength:    mergeconfig.MergeOptionalIntChooseLower(p1.MaxPayloadLength, p2.MaxPayloadLength),
		DropPartialPayloads: mergeconfig.MergeOptionalBools(p1.DropPartialPayloads, p2.DropPartialPayloads),
	}
}

func mergeDbPayloadCollectionRules(p1 *instrumentationrules.DbQueryPayloadCollection, p2 *instrumentationrules.DbQueryPayloadCollection) *instrumentationrules.DbQueryPayloadCollection {
	if p1 == nil {
		if p2 == nil {
			return nil
		}
		return p2.DeepCopy()
	}
	if p2 == nil {
		return p1.DeepCopy()
	}
	return &instrumentationrules.DbQueryPayloadCollection{
		MaxPayloadLength:    mergeconfig.MergeOptionalIntChooseLower(p1.MaxPayloadLength, p2.MaxPayloadLength),
		DropPartialPayloads: mergeconfig.MergeOptionalBools(p1.DropPartialPayloads, p2.DropPartialPayloads),
	}
}

func mergeMessagingPayloadCollectionRules(p1 *instrumentationrules.MessagingPayloadCollection, p2 *instrumentationrules.MessagingPayloadCollection) *instrumentationrules.MessagingPayloadCollection {
	if p1 == nil {
		if p2 == nil {
			return nil
		}
		return p2.DeepCopy()
	}
	if p2 == nil {
		return p1.DeepCopy()
	}
	return &instrumentationrules.MessagingPayloadCollection{
		MaxPayloadLength:    mergeconfig.MergeOptionalIntChooseLower(p1.MaxPayloadLength, p2.MaxPayloadLength),
		DropPartialPayloads: mergeconfig.MergeOptionalBools(p1.DropPartialPayloads, p2.DropPartialPayloads),
	}
}
