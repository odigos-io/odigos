import { ActionType, type Action } from '@odigos/ui-kit/types';
import { sortUrlTemplates } from './template-sort';
import { type ScopeToken, scopeTokensSortKey } from './scope-token-types';

export type { ScopeToken } from './scope-token-types';
export { scopeTokensSortKey } from './scope-token-types';

export type UrlTemplatizationDefaultGroup = {
  disabled?: boolean | null;
  scopes?: {
    namespaces?: string[] | null;
    languages?: string[] | null;
    sources?: { namespace?: string | null; kind?: string | null; name?: string | null }[] | null;
  } | null;
  skipPolicy?: {
    skipForNonSuccessCodes?: boolean | null;
    skipHttpStatusCodes?: number[] | null;
  } | null;
};

export type UrlTemplatizationRulesGroup = {
  filterK8sNamespace?: string | null;
  filterK8sWorkloadKind?: string | null;
  filterK8sWorkloadName?: string | null;
  filterProgrammingLanguage?: string | null;
  workloadFilters?: { kind?: string | null; name?: string | null }[] | null;
  templatizationRules?: { template?: string | null }[] | null;
  notes?: string | null;
};

export type UrlTemplatizationActionFields = {
  urlTemplatizationRulesGroups?: UrlTemplatizationRulesGroup[] | null;
  urlTemplatizationDefaultGroups?: UrlTemplatizationDefaultGroup[] | null;
};

export type AggregatedCustomTemplate = {
  template: string;
  scopeTokens: ScopeToken[];
  count: number;
  actionId: string;
  actionDisplayName?: string | null;
  actionLabels: string[];
  actionDisabled: boolean;
  actionUiGenerated: boolean;
};

export type CustomTemplateCrGroup = {
  actionId: string;
  actionDisplayName?: string | null;
  actionLabel: string;
  actionDisabled: boolean;
  actionUiGenerated: boolean;
  groupIndex: number;
  scopeTokens: ScopeToken[];
  groupNotes?: string | null;
  templates: string[];
};

export type AggregatedDefaultGroup = {
  scopeTokens: ScopeToken[];
  configLabel: string;
  count: number;
  actionLabels: string[];
  actionDisabled: boolean;
};

export type DefaultTemplatizationCrGroup = {
  actionId: string;
  actionLabel: string;
  actionDisabled: boolean;
  actionUiGenerated: boolean;
  groupIndex: number;
  scopeTokens: ScopeToken[];
  disabled: boolean;
  skipForNonSuccessCodes: boolean;
  skipHttpStatusCodes: number[];
  configLabel: string;
};

export type UrlTemplatizationAggregation = {
  customTemplates: AggregatedCustomTemplate[];
  defaultGroups: AggregatedDefaultGroup[];
  totalCustomRuleInstances: number;
  totalDefaultGroupInstances: number;
  actionCount: number;
  clusterDefaultEffect: ClusterDefaultTemplatizationEffect;
};

export type ClusterDefaultTemplatizationMode = 'none' | 'cluster_wide' | 'scoped';

export type ClusterDefaultTemplatizationEffect = {
  mode: ClusterDefaultTemplatizationMode;
  title: string;
  description: string;
  sectionSummary: string;
};

export function isUrlTemplatizationAction(action: Action): boolean {
  if (action.type === ActionType.URLTemplatization) {
    return true;
  }
  const fields = action.fields as UrlTemplatizationActionFields | undefined;
  const customGroups = fields?.urlTemplatizationRulesGroups;
  const defaultGroups = fields?.urlTemplatizationDefaultGroups;
  return (Array.isArray(customGroups) && customGroups.length > 0) || (Array.isArray(defaultGroups) && defaultGroups.length > 0);
}

function actionLabel(action: Action): string {
  return action.name?.trim() || action.id;
}

function actionDisplayName(action: Action): string | null {
  const name = action.name?.trim();
  return name || null;
}

function actionUiGenerated(action: Action): boolean {
  return !!(action as Action & { uiGenerated?: boolean | null }).uiGenerated;
}

export function isGlobalDefaultScope(scopes: UrlTemplatizationDefaultGroup['scopes']): boolean {
  if (!scopes) {
    return true;
  }
  const hasNamespaces = (scopes.namespaces?.length ?? 0) > 0;
  const hasSources = (scopes.sources?.length ?? 0) > 0;
  const hasLanguages = (scopes.languages?.length ?? 0) > 0;
  return !hasNamespaces && !hasSources && !hasLanguages;
}

function collectEnabledDefaultGroups(actions: Action[]): UrlTemplatizationDefaultGroup[] {
  const groups: UrlTemplatizationDefaultGroup[] = [];
  for (const action of actions) {
    if (action.disabled) {
      continue;
    }
    const fields = action.fields as UrlTemplatizationActionFields | undefined;
    for (const group of fields?.urlTemplatizationDefaultGroups ?? []) {
      groups.push(group);
    }
  }
  return groups;
}

export function deriveClusterDefaultTemplatizationEffect(actions: Action[]): ClusterDefaultTemplatizationEffect {
  const defaultGroups = collectEnabledDefaultGroups(actions);

  if (defaultGroups.length === 0) {
    return {
      mode: 'none',
      title: 'No default templatization rules',
      sectionSummary: 'No explicit default templating rules configured',
      description:
        'No enabled URL templatization action defines default templatization. Odigos still applies built-in heuristic default templatization for instrumented workloads covered by these actions.',
    };
  }

  const globalGroups = defaultGroups.filter((group) => isGlobalDefaultScope(group.scopes));
  if (globalGroups.length > 0) {
    const allGlobalDisabled = globalGroups.every((group) => group.disabled);
    if (allGlobalDisabled) {
      return {
        mode: 'cluster_wide',
        title: 'Default templatization disabled cluster-wide',
        sectionSummary: 'Default templating is disabled in the entire cluster',
        description:
          'At least one enabled action disables default templatization for all sources. Scoped default rules may still apply where configured.',
      };
    }

    const skipDescriptions = globalGroups
      .filter((group) => !group.disabled)
      .map((group) => formatDefaultConfig(group))
      .filter((label) => label !== 'Default heuristic templatization enabled');

    const skipDetail =
      skipDescriptions.length > 0
        ? ` ${[...new Set(skipDescriptions)].join('; ')}.`
        : ' Default heuristic templatization applies when no custom template matches.';

    return {
      mode: 'cluster_wide',
      title: 'Default templatization for the entire cluster',
      sectionSummary: 'Default templating is enabled in the entire cluster',
      description: `An enabled action configures default templatization for all sources.${skipDetail}`,
    };
  }

  return {
    mode: 'scoped',
    title: 'Default templatization for specific scopes only',
    sectionSummary: 'Default templatization for specific scopes only',
    description:
      'Enabled actions define default templatization only for selected namespaces, workloads, or languages—not cluster-wide. Workloads outside those scopes rely on built-in default behavior unless another rule applies.',
  };
}

export function customTemplatesSectionSummary(ruleCount: number): string {
  if (ruleCount === 0) {
    return 'No custom rules';
  }
  if (ruleCount === 1) {
    return '1 custom rule';
  }
  return `${ruleCount} custom rules`;
}

function formatLanguageToken(language: string): string {
  return language.trim().toLowerCase();
}

type WorkloadRef = {
  namespace?: string | null;
  kind?: string | null;
  name?: string | null;
};

function formatWorkloadKindLabel(kind: string | null | undefined): string {
  const trimmed = kind?.trim();
  return trimmed || 'Workload';
}

function formatWorkloadNamespaceName(namespace: string | null | undefined, name: string | null | undefined): string {
  const workloadName = name?.trim();
  if (!workloadName) {
    return '';
  }
  const ns = namespace?.trim();
  return ns ? `${ns}/${workloadName}` : workloadName;
}

function buildWorkloadScopeToken(workloads: WorkloadRef[], fallbackNamespace?: string): ScopeToken | null {
  const byKind = new Map<string, string[]>();

  for (const workload of workloads) {
    const kind = formatWorkloadKindLabel(workload.kind);
    const nsName = formatWorkloadNamespaceName(workload.namespace?.trim() || fallbackNamespace, workload.name);
    if (!nsName) {
      continue;
    }
    const list = byKind.get(kind) ?? [];
    list.push(nsName);
    byKind.set(kind, list);
  }

  if (byKind.size === 0) {
    return null;
  }

  const groups = [...byKind.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([kind, namespaceNames]) => ({ kind, namespaceNames }));

  return { type: 'workload', groups };
}

export function scopeTokensFromSourcesScopes(scopes: UrlTemplatizationDefaultGroup['scopes']): ScopeToken[] {
  if (isGlobalDefaultScope(scopes)) {
    return [{ type: 'entire_cluster' }];
  }

  const tokens: ScopeToken[] = [];

  if (scopes?.namespaces?.length) {
    tokens.push({ type: 'namespace', values: [...scopes.namespaces] });
  }

  if (scopes?.sources?.length) {
    const fallbackNamespace = scopes.namespaces?.length === 1 ? scopes.namespaces[0]?.trim() : undefined;
    const workloadToken = buildWorkloadScopeToken(
      scopes.sources.map((source) => ({
        namespace: source.namespace,
        kind: source.kind,
        name: source.name,
      })),
      fallbackNamespace,
    );
    if (workloadToken) {
      tokens.push(workloadToken);
    }
  }

  if (scopes?.languages?.length) {
    tokens.push({ type: 'language', values: scopes.languages.map(formatLanguageToken) });
  }

  return tokens.length ? tokens : [{ type: 'entire_cluster' }];
}

export function scopeTokensFromCustomRulesGroup(group: UrlTemplatizationRulesGroup): ScopeToken[] {
  const namespace = group.filterK8sNamespace?.trim();
  const language = group.filterProgrammingLanguage?.trim();
  const workloadFilters = group.workloadFilters?.filter((filter) => filter?.kind || filter?.name) ?? [];

  const hasLegacyWorkload = group.filterK8sWorkloadKind || group.filterK8sWorkloadName;
  if (!namespace && !language && workloadFilters.length === 0 && !hasLegacyWorkload) {
    return [{ type: 'entire_cluster' }];
  }

  const tokens: ScopeToken[] = [];

  if (namespace) {
    tokens.push({ type: 'namespace', values: [namespace] });
  }

  if (workloadFilters.length) {
    const workloadToken = buildWorkloadScopeToken(
      workloadFilters.map((filter) => ({
        namespace,
        kind: filter.kind,
        name: filter.name,
      })),
    );
    if (workloadToken) {
      tokens.push(workloadToken);
    }
  } else if (hasLegacyWorkload) {
    const workloadToken = buildWorkloadScopeToken([
      {
        namespace,
        kind: group.filterK8sWorkloadKind,
        name: group.filterK8sWorkloadName,
      },
    ]);
    if (workloadToken) {
      tokens.push(workloadToken);
    }
  }

  if (language) {
    tokens.push({ type: 'language', values: [formatLanguageToken(language)] });
  }

  return tokens;
}

function formatDefaultConfig(group: UrlTemplatizationDefaultGroup): string {
  if (group.disabled) {
    return 'Default templatization disabled';
  }
  const skip = group.skipPolicy;
  if (!skip) {
    return 'Default heuristic templatization enabled';
  }
  if (skip.skipForNonSuccessCodes) {
    return 'Skip default templatization for non-2xx responses';
  }
  if (skip.skipHttpStatusCodes?.length) {
    return `Skip default templatization for HTTP ${skip.skipHttpStatusCodes.join(', ')}`;
  }
  return 'Default templatization skip policy configured';
}

function defaultGroupKey(group: UrlTemplatizationDefaultGroup): string {
  const scopeTokens = scopeTokensFromSourcesScopes(group.scopes);
  return JSON.stringify({
    scopeTokens,
    config: formatDefaultConfig(group),
    disabled: !!group.disabled,
    skipForNonSuccess: !!group.skipPolicy?.skipForNonSuccessCodes,
    skipCodes: group.skipPolicy?.skipHttpStatusCodes ?? [],
  });
}

function customRuleKey(template: string, scopeTokens: ScopeToken[], actionId: string): string {
  return JSON.stringify({ template, scopeTokens, actionId });
}

export function collectCustomTemplateCrGroups(actions: Action[]): CustomTemplateCrGroup[] {
  const groups: CustomTemplateCrGroup[] = [];

  for (const action of actions) {
    if (!isUrlTemplatizationAction(action)) {
      continue;
    }
    const label = actionLabel(action);
    const fields = action.fields as UrlTemplatizationActionFields | undefined;

    for (const [groupIndex, group] of (fields?.urlTemplatizationRulesGroups ?? []).entries()) {
      const templates = sortUrlTemplates(
        (group.templatizationRules ?? [])
          .map((rule) => rule.template?.trim())
          .filter((template): template is string => !!template),
      );
      groups.push({
        actionId: action.id,
        actionDisplayName: actionDisplayName(action),
        actionLabel: label,
        actionDisabled: !!action.disabled,
        actionUiGenerated: actionUiGenerated(action),
        groupIndex,
        scopeTokens: scopeTokensFromCustomRulesGroup(group),
        groupNotes: group.notes,
        templates,
      });
    }
  }

  return groups;
}

export function collectDefaultTemplatizationCrGroups(actions: Action[]): DefaultTemplatizationCrGroup[] {
  const groups: DefaultTemplatizationCrGroup[] = [];

  for (const action of actions) {
    if (!isUrlTemplatizationAction(action)) {
      continue;
    }
    const label = actionLabel(action);
    const fields = action.fields as UrlTemplatizationActionFields | undefined;

    for (const [groupIndex, group] of (fields?.urlTemplatizationDefaultGroups ?? []).entries()) {
      groups.push({
        actionId: action.id,
        actionLabel: label,
        actionDisabled: !!action.disabled,
        actionUiGenerated: actionUiGenerated(action),
        groupIndex,
        scopeTokens: scopeTokensFromSourcesScopes(group.scopes),
        disabled: !!group.disabled,
        skipForNonSuccessCodes: !!group.skipPolicy?.skipForNonSuccessCodes,
        skipHttpStatusCodes: group.skipPolicy?.skipHttpStatusCodes ?? [],
        configLabel: formatDefaultConfig(group),
      });
    }
  }

  return groups;
}

export function aggregateUrlTemplatization(actions: Action[]): UrlTemplatizationAggregation {
  const customMap = new Map<
    string,
    {
      template: string;
      scopeTokens: ScopeToken[];
      count: number;
      actionLabels: Set<string>;
      actionId: string;
      actionDisplayName: string | null;
      actionDisabled: boolean;
      actionUiGenerated: boolean;
    }
  >();
  const defaultMap = new Map<
    string,
    {
      scopeTokens: ScopeToken[];
      configLabel: string;
      count: number;
      actionLabels: Set<string>;
      disabledActionLabels: Set<string>;
    }
  >();

  let totalCustomRuleInstances = 0;
  let totalDefaultGroupInstances = 0;

  for (const action of actions) {
    const label = actionLabel(action);
    const fields = action.fields as UrlTemplatizationActionFields | undefined;

    for (const group of fields?.urlTemplatizationRulesGroups ?? []) {
      const scopeTokens = scopeTokensFromCustomRulesGroup(group);
      for (const rule of group.templatizationRules ?? []) {
        const template = rule.template?.trim();
        if (!template) {
          continue;
        }
        totalCustomRuleInstances += 1;
        const key = customRuleKey(template, scopeTokens, action.id);
        const entry =
          customMap.get(key) ??
          ({
            template,
            scopeTokens,
            count: 0,
            actionLabels: new Set<string>(),
            actionId: action.id,
            actionDisplayName: actionDisplayName(action),
            actionDisabled: !!action.disabled,
            actionUiGenerated: actionUiGenerated(action),
          } as {
            template: string;
            scopeTokens: ScopeToken[];
            count: number;
            actionLabels: Set<string>;
            actionId: string;
            actionDisplayName: string | null;
            actionDisabled: boolean;
            actionUiGenerated: boolean;
          });
        entry.count += 1;
        entry.actionLabels.add(label);
        customMap.set(key, entry);
      }
    }

    for (const group of fields?.urlTemplatizationDefaultGroups ?? []) {
      totalDefaultGroupInstances += 1;
      const scopeTokens = scopeTokensFromSourcesScopes(group.scopes);
      const key = defaultGroupKey(group);
      const configLabel = formatDefaultConfig(group);
      const entry =
        defaultMap.get(key) ??
        ({
          scopeTokens,
          configLabel,
          count: 0,
          actionLabels: new Set<string>(),
          disabledActionLabels: new Set<string>(),
        } as {
          scopeTokens: ScopeToken[];
          configLabel: string;
          count: number;
          actionLabels: Set<string>;
          disabledActionLabels: Set<string>;
        });
      entry.count += 1;
      entry.actionLabels.add(label);
      if (action.disabled) {
        entry.disabledActionLabels.add(label);
      }
      defaultMap.set(key, entry);
    }
  }

  const customTemplates = [...customMap.values()]
    .map(
      ({
        template,
        scopeTokens,
        count,
        actionLabels,
        actionId,
        actionDisplayName,
        actionDisabled,
        actionUiGenerated,
      }) => ({
        template,
        scopeTokens,
        count,
        actionId,
        actionDisplayName,
        actionLabels: [...actionLabels].sort((a, b) => a.localeCompare(b)),
        actionDisabled,
        actionUiGenerated,
      }),
    )
    .sort(
      (a, b) =>
        a.template.localeCompare(b.template) ||
        scopeTokensSortKey(a.scopeTokens).localeCompare(scopeTokensSortKey(b.scopeTokens)) ||
        a.actionLabels.join(',').localeCompare(b.actionLabels.join(',')),
    );

  const defaultGroups = [...defaultMap.values()]
    .map(({ scopeTokens, configLabel, count, actionLabels, disabledActionLabels }) => ({
      scopeTokens,
      configLabel,
      count,
      actionLabels: [...actionLabels].sort((a, b) => a.localeCompare(b)),
      actionDisabled:
        actionLabels.size > 0 && disabledActionLabels.size === actionLabels.size,
    }))
    .sort(
      (a, b) =>
        b.count - a.count ||
        scopeTokensSortKey(a.scopeTokens).localeCompare(scopeTokensSortKey(b.scopeTokens)) ||
        a.configLabel.localeCompare(b.configLabel),
    );

  return {
    customTemplates,
    defaultGroups,
    totalCustomRuleInstances,
    totalDefaultGroupInstances,
    actionCount: actions.length,
    clusterDefaultEffect: deriveClusterDefaultTemplatizationEffect(actions),
  };
}
