package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
)

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

func mergePayloadCollectionConfigs(agg *instrumentationrules.PayloadCollection, new *instrumentationrules.PayloadCollection) *instrumentationrules.PayloadCollection {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	if new.HttpRequest != nil {
		agg.HttpRequest = mergeHttpPayloadCollectionRules(agg.HttpRequest, new.HttpRequest)
	}
	if new.HttpResponse != nil {
		agg.HttpResponse = mergeHttpPayloadCollectionRules(agg.HttpResponse, new.HttpResponse)
	}
	if new.DbQuery != nil {
		agg.DbQuery = mergeDbPayloadCollectionRules(agg.DbQuery, new.DbQuery)
	}
	if new.Messaging != nil {
		agg.Messaging = mergeMessagingPayloadCollectionRules(agg.Messaging, new.Messaging)
	}
	return agg
}

func mergeHttpPayloadCollectionRules(agg *instrumentationrules.HttpPayloadCollection, new *instrumentationrules.HttpPayloadCollection) *instrumentationrules.HttpPayloadCollection {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	return &instrumentationrules.HttpPayloadCollection{
		MimeTypes:           mergeMimeTypeMap(agg.MimeTypes, new.MimeTypes),
		MaxPayloadLength:    mergeMaxPayloadLength(agg.MaxPayloadLength, new.MaxPayloadLength),
		DropPartialPayloads: mergeDropPartialPayloads(agg.DropPartialPayloads, new.DropPartialPayloads),
	}
}

func mergeDbPayloadCollectionRules(agg *instrumentationrules.DbQueryPayloadCollection, new *instrumentationrules.DbQueryPayloadCollection) *instrumentationrules.DbQueryPayloadCollection {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	return &instrumentationrules.DbQueryPayloadCollection{
		MaxPayloadLength:    mergeMaxPayloadLength(agg.MaxPayloadLength, new.MaxPayloadLength),
		DropPartialPayloads: mergeDropPartialPayloads(agg.DropPartialPayloads, new.DropPartialPayloads),
	}
}

func mergeMessagingPayloadCollectionRules(agg *instrumentationrules.MessagingPayloadCollection, new *instrumentationrules.MessagingPayloadCollection) *instrumentationrules.MessagingPayloadCollection {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	return &instrumentationrules.MessagingPayloadCollection{
		MaxPayloadLength:    mergeMaxPayloadLength(agg.MaxPayloadLength, new.MaxPayloadLength),
		DropPartialPayloads: mergeDropPartialPayloads(agg.DropPartialPayloads, new.DropPartialPayloads),
	}
}

func mergeDropPartialPayloads(agg *bool, new *bool) *bool {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	if *agg {
		return agg
	} else if *new {
		return new
	} else {
		f := false
		return &f
	}
}

func mergeMaxPayloadLength(agg *int64, new *int64) *int64 {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	if *agg < *new {
		return agg
	} else {
		return new
	}
}

func mergeMimeTypeMap(agg *[]string, new *[]string) *[]string {
	if agg == nil {
		return new
	}
	if new == nil {
		return agg
	}
	allMimes := map[string]struct{}{}
	for _, mime := range *agg {
		allMimes[mime] = struct{}{}
	}
	for _, mime := range *new {
		allMimes[mime] = struct{}{}
	}
	mergedMimes := make([]string, 0, len(allMimes))
	for mime := range allMimes {
		mergedMimes = append(mergedMimes, mime)
	}
	return &mergedMimes
}
