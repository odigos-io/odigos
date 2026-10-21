package insights

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

func DeliverySettingsToModel(value DeliverySettings) *model.InsightsDeliverySettings {
	return &model.InsightsDeliverySettings{
		Muted:                  value.Muted,
		ExcludedIntegrationIds: append([]string{}, value.ExcludedIntegrationIds...),
	}
}

func IntegrationFieldSpecToModel(value IntegrationFieldSpec) *model.InsightsIntegrationFieldSpec {
	return &model.InsightsIntegrationFieldSpec{
		Key:         value.Key,
		Label:       value.Label,
		Type:        model.InsightsIntegrationFieldType(value.Type),
		Required:    value.Required,
		Help:        value.Help,
		Placeholder: value.Placeholder,
		Default:     value.Default,
		Options:     append([]string{}, value.Options...),
	}
}

func IntegrationCapabilitiesToModel(value IntegrationCapabilities) *model.InsightsIntegrationCapabilities {
	return &model.InsightsIntegrationCapabilities{
		NeedsHeartbeat:    value.NeedsHeartbeat,
		SupportsResolve:   value.SupportsResolve,
		ReturnsExternalID: value.ReturnsExternalID,
		MaxBatch:          value.MaxBatch,
	}
}

func IntegrationDescriptorToModel(value IntegrationDescriptor) *model.InsightsIntegrationDescriptor {
	return &model.InsightsIntegrationDescriptor{
		Type:         value.Type,
		Name:         value.Name,
		Description:  value.Description,
		DocsURL:      value.DocsURL,
		Fields:       mapSlice(value.Fields, IntegrationFieldSpecToModel),
		Capabilities: IntegrationCapabilitiesToModel(value.Capabilities),
	}
}

func IntegrationFiltersToModel(value IntegrationFilters) *model.InsightsIntegrationFilters {
	var severity *model.InsightsSeverity
	if value.MinSeverity != nil {
		v := SeverityToModel(*value.MinSeverity)
		severity = &v
	}
	return &model.InsightsIntegrationFilters{
		MinSeverity:     severity,
		Kinds:           mapSlice(value.Kinds, FindingKindToModel),
		NotifyOnResolve: value.NotifyOnResolve,
	}
}

// emptyIfZeroTimestamp maps the engine's Go zero time to "" for the non-null
// GraphQL String fields, so clients can tell "never happened" from a real date.
func emptyIfZeroTimestamp(timestamp string) string {
	if isZeroTimestamp(timestamp) {
		return ""
	}
	return timestamp
}

func IntegrationHealthToModel(value IntegrationHealth) *model.InsightsIntegrationHealth {
	return &model.InsightsIntegrationHealth{
		LastSuccessAt:       emptyIfZeroTimestamp(value.LastSuccessAt),
		LastFailureAt:       emptyIfZeroTimestamp(value.LastFailureAt),
		LastError:           value.LastError,
		ConsecutiveFailures: value.ConsecutiveFailures,
		Pending:             value.Pending,
		Failed:              value.Failed,
		Dead:                value.Dead,
	}
}

func IntegrationToModel(value Integration) *model.InsightsIntegration {
	return &model.InsightsIntegration{
		ID:        value.ID,
		Name:      value.Name,
		Type:      value.Type,
		Enabled:   value.Enabled,
		Config:    string(value.Config),
		Filters:   IntegrationFiltersToModel(value.Filters),
		Health:    IntegrationHealthToModel(value.Health),
		CreatedAt: value.CreatedAt,
		UpdatedAt: value.UpdatedAt,
	}
}

func DeliveryToModel(value Delivery) *model.InsightsDelivery {
	return &model.InsightsDelivery{
		IntegrationID:     value.IntegrationID,
		FindingKey:        value.FindingKey,
		LastAction:        model.InsightsDeliveryAction(value.LastAction),
		State:             model.InsightsDeliveryState(value.State),
		Attempts:          value.Attempts,
		NextAttemptAt:     emptyIfZeroTimestamp(value.NextAttemptAt),
		LastError:         value.LastError,
		SentAt:            emptyIfZeroTimestamp(value.SentAt),
		OccurrencesAtSend: value.OccurrencesAtSend,
		SeverityAtSend:    value.SeverityAtSend,
		StatusAtSend:      value.StatusAtSend,
		ExternalID:        value.ExternalID,
		UpdatedAt:         value.UpdatedAt,
	}
}

func DeliveryAttemptToModel(value DeliveryAttempt) *model.InsightsDeliveryAttempt {
	return &model.InsightsDeliveryAttempt{
		IntegrationID: value.IntegrationID,
		FindingKey:    value.FindingKey,
		Action:        model.InsightsDeliveryAction(value.Action),
		Attempt:       value.Attempt,
		Outcome:       model.InsightsDeliveryAttemptOutcome(value.Outcome),
		HTTPStatus:    value.HTTPStatus,
		Error:         value.Error,
		DurationMs:    value.DurationMs,
		EventCount:    value.EventCount,
		AttemptedAt:   value.AttemptedAt,
	}
}

func DeliverySettingsFromInput(value *model.InsightsDeliverySettingsInput) *DeliverySettings {
	if value == nil {
		return nil
	}
	return &DeliverySettings{Muted: value.Muted, ExcludedIntegrationIds: value.ExcludedIntegrationIds}
}
func OptionalDeliverySettingsToModel(value *DeliverySettings) *model.InsightsDeliverySettings {
	if value == nil {
		return nil
	}
	return DeliverySettingsToModel(*value)
}
func IntegrationFromInput(input model.InsightsIntegrationInput) (IntegrationWrite, error) {
	var config map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input.Config), &config); err != nil || config == nil {
		return IntegrationWrite{}, fmt.Errorf("%w: integration config must be a JSON object", ErrBadRequest)
	}
	filters := IntegrationFilters{Kinds: mapSlice(input.Filters.Kinds, func(value model.InsightsFindingKind) FindingKind { return FindingKind(value) }), NotifyOnResolve: input.Filters.NotifyOnResolve}
	if input.Filters.MinSeverity != nil {
		v := Severity(*input.Filters.MinSeverity)
		filters.MinSeverity = &v
	}
	return IntegrationWrite{Name: input.Name, Type: input.Type, Enabled: input.Enabled, Config: json.RawMessage(input.Config), Filters: filters}, nil
}
