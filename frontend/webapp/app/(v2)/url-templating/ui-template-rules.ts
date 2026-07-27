import type { Action } from '@odigos/ui-kit/types';
import type { ScopeToken } from './scope-token-types';
import type { CustomTemplateCrGroup } from './url-templatization-aggregate';

/** Default Action name for rules created from the URL Templating UI. */
export const UI_TEMPLATE_RULES_ACTION_NAME = 'UI template rules';

export function findUiTemplateRulesAction(actions: Action[]): Action | undefined {
  const target = UI_TEMPLATE_RULES_ACTION_NAME.toLowerCase();
  return actions.find((action) => action.name?.trim().toLowerCase() === target);
}

export function isEntireClusterScopeTokens(tokens: ScopeToken[]): boolean {
  return tokens.length === 1 && tokens[0]?.type === 'entire_cluster';
}

export function isEntireClusterGroup(group: CustomTemplateCrGroup): boolean {
  return isEntireClusterScopeTokens(group.scopeTokens);
}

export function findEntireClusterGroupForAction(
  groups: CustomTemplateCrGroup[],
  actionId: string,
): CustomTemplateCrGroup | undefined {
  return groups.find((group) => group.actionId === actionId && isEntireClusterGroup(group));
}

function scopeTokenSearchParts(token: ScopeToken): string[] {
  switch (token.type) {
    case 'entire_cluster':
      return ['entire cluster', 'cluster', 'all'];
    case 'namespace':
      return ['namespace', ...token.values];
    case 'language':
      return ['language', ...token.values];
    case 'workload':
      return [
        'workload',
        ...token.groups.flatMap((group) => [group.kind, ...group.namespaceNames]),
      ];
    default:
      return [];
  }
}

export function customTemplateGroupSearchText(group: CustomTemplateCrGroup): string {
  return [
    group.actionLabel,
    group.actionDisplayName,
    ...group.scopeTokens.flatMap(scopeTokenSearchParts),
    ...group.templates,
    `${group.templates.length} templates`,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();
}
