import type { Action, SourcesScopes } from '@odigos/ui-kit/types';
import { K8sResourceKind } from '@odigos/ui-kit/types';
import type {
  UrlTemplatizationActionFields,
  UrlTemplatizationDefaultGroup,
} from './url-templatization-aggregate';
import { makeEmptySourcesScopes } from './action-group-templates-update';

export type DefaultGroupSkipPolicyInput = {
  skipForNonSuccessCodes?: boolean | null;
  skipHttpStatusCodes?: number[] | null;
};

export type DefaultGroupEditInput = {
  scopes: SourcesScopes;
  disabled: boolean;
  skipForNonSuccessCodes: boolean;
  skipHttpStatusCodes: number[];
};

export type UpdateActionWithDefaultGroupsVars = {
  id: string;
  action: {
    type: Action['type'];
    name?: string | null;
    notes?: string | null;
    disabled?: boolean | null;
    signals?: Action['signals'];
    fields: {
      urlTemplatizationDefaultGroups: UrlTemplatizationDefaultGroup[];
    };
  };
};

function parseWorkloadKind(kind: string | null | undefined): K8sResourceKind {
  const trimmed = kind?.trim();
  if (!trimmed) {
    return K8sResourceKind.Deployment;
  }
  const values = Object.values(K8sResourceKind) as string[];
  if (values.includes(trimmed)) {
    return trimmed as K8sResourceKind;
  }
  return K8sResourceKind.Deployment;
}

export function defaultGroupScopesToSourcesScopes(
  scopes: UrlTemplatizationDefaultGroup['scopes'],
): SourcesScopes {
  if (!scopes) {
    return makeEmptySourcesScopes();
  }
  return {
    namespaces: [...(scopes.namespaces ?? [])],
    languages: [...(scopes.languages ?? [])],
    sources: (scopes.sources ?? [])
      .filter((source) => source?.name?.trim())
      .map((source) => ({
        namespace: source.namespace?.trim() || '',
        kind: parseWorkloadKind(source.kind),
        name: source.name?.trim() || '',
      })),
  };
}

export function sourcesScopesToDefaultGroupScopes(
  scopes: SourcesScopes,
): UrlTemplatizationDefaultGroup['scopes'] {
  const namespaces = (scopes.namespaces ?? []).map((value) => value.trim()).filter(Boolean);
  const languages = (scopes.languages ?? []).map((value) => value.trim()).filter(Boolean);
  const sources = (scopes.sources ?? [])
    .filter((source) => source.name?.trim())
    .map((source) => ({
      namespace: source.namespace?.trim() || null,
      kind: source.kind || null,
      name: source.name?.trim() || null,
    }));

  if (!namespaces.length && !languages.length && !sources.length) {
    return null;
  }

  return {
    namespaces: namespaces.length ? namespaces : null,
    languages: languages.length ? languages : null,
    sources: sources.length ? sources : null,
  };
}

export function buildDefaultGroupFromEditInput(input: DefaultGroupEditInput): UrlTemplatizationDefaultGroup {
  const skipHttpStatusCodes = input.skipForNonSuccessCodes
    ? []
    : input.skipHttpStatusCodes.filter((code) => Number.isFinite(code) && code > 0);

  const hasSkipPolicy = input.skipForNonSuccessCodes || skipHttpStatusCodes.length > 0;

  return {
    scopes: sourcesScopesToDefaultGroupScopes(input.scopes),
    disabled: input.disabled,
    skipPolicy: hasSkipPolicy
      ? {
          skipForNonSuccessCodes: input.skipForNonSuccessCodes || null,
          skipHttpStatusCodes: skipHttpStatusCodes.length ? skipHttpStatusCodes : null,
        }
      : null,
  };
}

function actionUpdateEnvelope(
  action: Action,
  groups: UrlTemplatizationDefaultGroup[],
): UpdateActionWithDefaultGroupsVars {
  return {
    id: action.id,
    action: {
      type: action.type,
      name: action.name,
      notes: action.notes,
      disabled: action.disabled,
      signals: action.signals,
      fields: {
        urlTemplatizationDefaultGroups: groups,
      },
    },
  };
}

function cloneDefaultGroups(action: Action): UrlTemplatizationDefaultGroup[] {
  const fields = (action.fields ?? {}) as UrlTemplatizationActionFields;
  return [...(fields.urlTemplatizationDefaultGroups ?? [])].map((group) => ({
    scopes: group.scopes
      ? {
          namespaces: group.scopes.namespaces ? [...group.scopes.namespaces] : null,
          languages: group.scopes.languages ? [...group.scopes.languages] : null,
          sources: group.scopes.sources
            ? group.scopes.sources.map((source) => ({ ...source }))
            : null,
        }
      : null,
    disabled: group.disabled ?? null,
    skipPolicy: group.skipPolicy
      ? {
          skipForNonSuccessCodes: group.skipPolicy.skipForNonSuccessCodes ?? null,
          skipHttpStatusCodes: group.skipPolicy.skipHttpStatusCodes
            ? [...group.skipPolicy.skipHttpStatusCodes]
            : null,
        }
      : null,
  }));
}

export function buildUpdateActionWithDefaultGroup(
  action: Action,
  groupIndex: number,
  input: DefaultGroupEditInput,
): UpdateActionWithDefaultGroupsVars {
  const groups = cloneDefaultGroups(action);
  if (groupIndex < 0 || groupIndex >= groups.length) {
    throw new Error(`Default group index ${groupIndex} is out of range for action ${action.id}`);
  }
  groups[groupIndex] = buildDefaultGroupFromEditInput(input);
  return actionUpdateEnvelope(action, groups);
}

export function buildUpdateActionWithDefaultGroupDeleted(
  action: Action,
  groupIndex: number,
): UpdateActionWithDefaultGroupsVars {
  const groups = cloneDefaultGroups(action);
  if (groupIndex < 0 || groupIndex >= groups.length) {
    throw new Error(`Default group index ${groupIndex} is out of range for action ${action.id}`);
  }
  groups.splice(groupIndex, 1);
  return actionUpdateEnvelope(action, groups);
}

export const CLUSTER_WIDE_ENABLED_DEFAULT_GROUP: UrlTemplatizationDefaultGroup = {
  scopes: null,
  disabled: false,
  skipPolicy: null,
};

export function buildUpdateActionWithAppendedDefaultGroup(
  action: Action,
  group: UrlTemplatizationDefaultGroup = CLUSTER_WIDE_ENABLED_DEFAULT_GROUP,
): UpdateActionWithDefaultGroupsVars {
  const groups = cloneDefaultGroups(action);
  groups.push(group);
  return actionUpdateEnvelope(action, groups);
}

/** Re-enable every cluster-wide default group that is currently disabled on this action. */
export function buildUpdateActionReenablingClusterWideDefaults(
  action: Action,
): UpdateActionWithDefaultGroupsVars | null {
  const groups = cloneDefaultGroups(action);
  let changed = false;
  for (const group of groups) {
    if (isGlobalDefaultScope(group.scopes) && group.disabled) {
      group.disabled = false;
      changed = true;
    }
  }
  if (!changed) {
    return null;
  }
  return actionUpdateEnvelope(action, groups);
}

function isGlobalDefaultScope(scopes: UrlTemplatizationDefaultGroup['scopes']): boolean {
  if (!scopes) {
    return true;
  }
  const hasNamespaces = (scopes.namespaces?.length ?? 0) > 0;
  const hasSources = (scopes.sources?.length ?? 0) > 0;
  const hasLanguages = (scopes.languages?.length ?? 0) > 0;
  return !hasNamespaces && !hasSources && !hasLanguages;
}

export function parseSkipHttpStatusCodesInput(value: string): { ok: true; codes: number[] } | { ok: false; error: string } {
  const trimmed = value.trim();
  if (!trimmed) {
    return { ok: true, codes: [] };
  }

  const parts = trimmed.split(/[,\s]+/).map((part) => part.trim()).filter(Boolean);
  const codes: number[] = [];
  const seen = new Set<number>();

  for (const part of parts) {
    if (!/^\d{3}$/.test(part)) {
      return { ok: false, error: `Invalid HTTP status code "${part}". Use 3-digit codes like 404.` };
    }
    const code = Number(part);
    if (code < 100 || code > 599) {
      return { ok: false, error: `HTTP status code ${code} is out of range (100–599).` };
    }
    if (!seen.has(code)) {
      seen.add(code);
      codes.push(code);
    }
  }

  return { ok: true, codes };
}

export function formatSkipHttpStatusCodesInput(codes: number[]): string {
  return codes.join(', ');
}

export function defaultGroupEditInputsEqual(a: DefaultGroupEditInput, b: DefaultGroupEditInput): boolean {
  return JSON.stringify(normalizeEditInput(a)) === JSON.stringify(normalizeEditInput(b));
}

function normalizeEditInput(input: DefaultGroupEditInput): unknown {
  return {
    scopes: {
      namespaces: [...input.scopes.namespaces].map((v) => v.trim()).filter(Boolean).sort(),
      languages: [...input.scopes.languages].map((v) => v.trim()).filter(Boolean).sort(),
      sources: [...input.scopes.sources]
        .map((source) => ({
          namespace: source.namespace?.trim() || '',
          kind: source.kind || '',
          name: source.name?.trim() || '',
        }))
        .filter((source) => source.name)
        .sort((left, right) =>
          `${left.namespace}/${left.kind}/${left.name}`.localeCompare(
            `${right.namespace}/${right.kind}/${right.name}`,
          ),
        ),
    },
    disabled: input.disabled,
    skipForNonSuccessCodes: input.skipForNonSuccessCodes,
    skipHttpStatusCodes: input.skipForNonSuccessCodes
      ? []
      : [...input.skipHttpStatusCodes].filter((code) => Number.isFinite(code) && code > 0).sort((x, y) => x - y),
  };
}
