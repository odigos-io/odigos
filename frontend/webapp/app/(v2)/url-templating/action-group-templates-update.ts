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

export function makeEmptySourcesScopes(): SourcesScopes {
  return { sources: [], namespaces: [], languages: [] };
}

/**
 * Fold SourceScopeSection's SourcesScopes into the GraphQL URL-templatization
 * filter form (single namespace / language + workloadFilters). Multi-value
 * scopes are rejected so we do not silently drop entries on convert.
 */
export function sourcesScopesToAppendRuleGroupFilters(
  scopes: SourcesScopes,
): { ok: true; filters: Omit<AppendRuleGroupInput, 'templates' | 'notes'> } | { ok: false; error: string } {
  const sources = scopes.sources ?? [];
  const namespaces = scopes.namespaces ?? [];
  const languages = scopes.languages ?? [];

  if (sources.length > 0) {
    const distinctNamespaces = Array.from(
      new Set(sources.map((source) => source.namespace?.trim()).filter(Boolean)),
    );
    if (distinctNamespaces.length > 1) {
      return {
        ok: false,
        error: 'Selected sources must share a single namespace for URL templatization rule groups.',
      };
    }
    if (namespaces.length > 0 || languages.length > 0) {
      return {
        ok: false,
        error: 'Choose either sources, namespaces, or languages — not a combination.',
      };
    }
    return {
      ok: true,
      filters: {
        filterK8sNamespace: distinctNamespaces[0] ?? null,
        filterProgrammingLanguage: null,
        workloadFilters: sources.map((source) => ({
          kind: source.kind || null,
          name: source.name || null,
        })),
      },
    };
  }

  if (namespaces.length > 0) {
    if (namespaces.length > 1) {
      return {
        ok: false,
        error: 'Select a single namespace for this rule group.',
      };
    }
    if (languages.length > 0) {
      return {
        ok: false,
        error: 'Choose either sources, namespaces, or languages — not a combination.',
      };
    }
    return {
      ok: true,
      filters: {
        filterK8sNamespace: namespaces[0]?.trim() || null,
        filterProgrammingLanguage: null,
        workloadFilters: null,
      },
    };
  }

  if (languages.length > 0) {
    if (languages.length > 1) {
      return {
        ok: false,
        error: 'Select a single programming language for this rule group.',
      };
    }
    return {
      ok: true,
      filters: {
        filterK8sNamespace: null,
        filterProgrammingLanguage: languages[0]?.trim() || null,
        workloadFilters: null,
      },
    };
  }

  return {
    ok: true,
    filters: {
      filterK8sNamespace: null,
      filterProgrammingLanguage: null,
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
