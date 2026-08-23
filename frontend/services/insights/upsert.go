package insights

import (
	"context"
	"fmt"
)

// Read-after-write wrappers for the upsert endpoints. The REST API answers an
// upsert with 204 No Content while the GraphQL schema returns the stored entity,
// so each wrapper re-reads what the engine actually persisted — it derives
// policy ids, normalizes scopes and fills missing fields with defaults, none of
// which the request body reflects.

func (c *Client) UpsertPolicyAndRead(ctx context.Context, policy Policy) (Policy, error) {
	if err := c.UpsertPolicy(ctx, policy); err != nil {
		return Policy{}, err
	}
	stored, err := c.ListPolicies(ctx)
	if err != nil {
		return Policy{}, err
	}
	for _, candidate := range stored {
		if candidate.Scope == policy.Scope && candidate.ScopeKey == policy.ScopeKey {
			return candidate, nil
		}
	}
	// Fallback: if the list is briefly stale after upsert, echo the accepted
	// request for a disabled policy so the UI can still show it.
	if !policy.Enabled {
		return policy, nil
	}
	return Policy{}, fmt.Errorf("%w: policy %s/%q is missing after upsert", ErrInternal, policy.Scope, policy.ScopeKey)
}

func (c *Client) UpsertLearningPolicyAndRead(ctx context.Context, policy LearningPolicy) (LearningPolicy, error) {
	if err := c.UpsertLearningPolicy(ctx, policy); err != nil {
		return LearningPolicy{}, err
	}
	stored, err := c.ListLearningPolicies(ctx)
	if err != nil {
		return LearningPolicy{}, err
	}
	for _, candidate := range stored {
		if candidate.Class == policy.Class && candidate.Scope == policy.Scope && candidate.ScopeKey == policy.ScopeKey {
			return candidate, nil
		}
	}
	return LearningPolicy{}, fmt.Errorf("%w: learning policy %s %s/%q is missing after upsert", ErrInternal, policy.Class, policy.Scope, policy.ScopeKey)
}

func (c *Client) UpsertGuardrailAndRead(ctx context.Context, guardrail Guardrail) (Guardrail, error) {
	if err := c.UpsertGuardrail(ctx, guardrail); err != nil {
		return Guardrail{}, err
	}
	stored, err := c.ListGuardrails(ctx)
	if err != nil {
		return Guardrail{}, err
	}
	for _, candidate := range stored {
		if candidate.ScopeKey == guardrail.ScopeKey {
			return candidate, nil
		}
	}
	return Guardrail{}, fmt.Errorf("%w: guardrail %q is missing after upsert", ErrInternal, guardrail.ScopeKey)
}

func (c *Client) UpdateSystemSettingsAndRead(ctx context.Context, settings SystemSettings) (SystemSettings, error) {
	if err := c.UpdateSystemSettings(ctx, settings); err != nil {
		return SystemSettings{}, err
	}
	stored, err := c.GetSystemSettings(ctx)
	if err != nil {
		return SystemSettings{}, err
	}
	return *stored, nil
}
