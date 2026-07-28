import type { Action, SourcesScopes } from '@odigos/ui-kit/types';
import type { UrlTemplatizationActionFields, UrlTemplatizationRulesGroup } from './url-templatization-aggregate';
import { sortUrlTemplates } from './template-sort';

export type UpdateActionWithGroupTemplatesVars = {
  id: string;
  action: {
    type: Action['type'];
    name?: string | null;
    notes?: string | null;
    disabled?: boolean | null;
    signals?: Action['signals'];
    fields: {
      urlTemplatizationRulesGroups: UrlTemplatizationRulesGroup[];
    };
  };
};

export type AppendRuleGroupInput = {
  filterK8sNamespace?: string | null;
  filterProgrammingLanguage?: string | null;
  workloadFilters?: { kind?: string | null; name?: string | null }[] | null;
  notes?: string | null;
  templates: string[];
};

/** Separates per-source namespaces from additional namespace-scope entries in filterK8sNamespace. */
export const RULE_GROUP_NAMESPACE_SCOPE_SEPARATOR = '||';

export function makeEmptySourcesScopes(): SourcesScopes {
  return { sources: [], namespaces: [], languages: [] };
}

function joinCsv(values: string[]): string | null {
  const cleaned = values.map((value) => value.trim()).filter(Boolean);
  return cleaned.length ? cleaned.join(',') : null;
}

/**
 * Map SourceScopeSection's SourcesScopes into the existing GraphQL filter fields.
 * Multi-value selections are encoded as CSV (and optional `||` for extra namespaces)
 * so the API shape stays unchanged while any number of entities is allowed.
 */
export function sourcesScopesToAppendRuleGroupFilters(
  scopes: SourcesScopes,
): { ok: true; filters: Omit<AppendRuleGroupInput, 'templates' | 'notes'> } {
  const sources = (scopes.sources ?? []).filter((source) => source.name?.trim());
  const namespaces = (scopes.namespaces ?? []).map((value) => value.trim()).filter(Boolean);
  const languages = (scopes.languages ?? []).map((value) => value.trim()).filter(Boolean);

  const filterProgrammingLanguage = joinCsv(languages);

  if (sources.length > 0) {
    // Keep empty slots so namespace indices stay aligned with workloadFilters.
    const sourceNamespaces = sources.map((source) => source.namespace?.trim() || '');
    const sourceNsCsv = sourceNamespaces.join(',');
    const extraNsCsv = joinCsv(namespaces);
    const filterK8sNamespace = extraNsCsv
      ? `${sourceNsCsv}${RULE_GROUP_NAMESPACE_SCOPE_SEPARATOR}${extraNsCsv}`
      : sourceNsCsv || null;

    return {
      ok: true,
      filters: {
        filterK8sNamespace,
        filterProgrammingLanguage,
        workloadFilters: sources.map((source) => ({
          kind: source.kind || null,
          name: source.name || null,
        })),
      },
    };
  }

  return {
    ok: true,
    filters: {
      filterK8sNamespace: joinCsv(namespaces),
      filterProgrammingLanguage,
      workloadFilters: null,
    },
  };
}

function rulesGroupWithTemplates(
  group: UrlTemplatizationRulesGroup,
  templateStrings: string[],
): UrlTemplatizationRulesGroup {
  const existingRules = group.templatizationRules ?? [];
  const templatizationRules = templateStrings.map((template) => {
    const trimmed = template.trim();
    const match = existingRules.find((rule) => rule.template?.trim() === trimmed);
    return match ? { ...match, template: trimmed } : { template: trimmed };
  });

  return {
    ...group,
    templatizationRules,
  };
}

function actionUpdateEnvelope(
  action: Action,
  groups: UrlTemplatizationRulesGroup[],
): UpdateActionWithGroupTemplatesVars {
  return {
    id: action.id,
    action: {
      type: action.type,
      name: action.name,
      notes: action.notes,
      disabled: action.disabled,
      signals: action.signals,
      fields: {
        urlTemplatizationRulesGroups: groups,
      },
    },
  };
}

export function buildUpdateActionWithGroupTemplates(
  action: Action,
  groupIndex: number,
  templateStrings: string[],
): UpdateActionWithGroupTemplatesVars {
  const fields = (action.fields ?? {}) as UrlTemplatizationActionFields;
  const groups = [...(fields.urlTemplatizationRulesGroups ?? [])];

  if (groupIndex < 0 || groupIndex >= groups.length) {
    throw new Error(`Rule group index ${groupIndex} is out of range for action ${action.id}`);
  }

  const normalizedTemplates = sortUrlTemplates(templateStrings.map((t) => t.trim()).filter(Boolean));
  groups[groupIndex] = rulesGroupWithTemplates(groups[groupIndex], normalizedTemplates);

  return actionUpdateEnvelope(action, groups);
}

export function buildUpdateActionWithAppendedGroup(
  action: Action,
  groupInput: AppendRuleGroupInput,
): UpdateActionWithGroupTemplatesVars {
  const fields = (action.fields ?? {}) as UrlTemplatizationActionFields;
  const groups = [...(fields.urlTemplatizationRulesGroups ?? [])];
  const normalizedTemplates = sortUrlTemplates(groupInput.templates.map((t) => t.trim()).filter(Boolean));

  const newGroup: UrlTemplatizationRulesGroup = {
    filterK8sNamespace: groupInput.filterK8sNamespace?.trim() || null,
    filterProgrammingLanguage: groupInput.filterProgrammingLanguage?.trim() || null,
    workloadFilters: groupInput.workloadFilters?.length ? groupInput.workloadFilters : null,
    notes: groupInput.notes?.trim() || null,
    templatizationRules: normalizedTemplates.map((template) => ({ template })),
  };

  groups.push(newGroup);
  return actionUpdateEnvelope(action, groups);
}
