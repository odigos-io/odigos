import type { ScopeToken } from './scope-token-types';
import { scopeTokensSortKey } from './scope-token-types';

export type CrTemplateGroupRef = {
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

export type UrlTemplateDetail = {
  template: string;
  scopeTokens: ScopeToken[];
  actionId: string;
  actionDisplayName?: string | null;
  actionLabel: string;
  actionDisabled: boolean;
  actionUiGenerated: boolean;
  count: number;
  groupIndex?: number;
  groupNotes?: string | null;
};

export function urlTemplateDetailKey(detail: UrlTemplateDetail): string {
  return `${detail.actionId}|${detail.template}|${detail.scopeTokens.map((t) => t.type).join(',')}|${detail.groupIndex ?? ''}`;
}

export function detailFromAggregatedTemplate(item: {
  template: string;
  scopeTokens: ScopeToken[];
  actionId: string;
  actionDisplayName?: string | null;
  actionLabels: string[];
  actionDisabled: boolean;
  actionUiGenerated: boolean;
  count: number;
}): UrlTemplateDetail {
  return {
    template: item.template,
    scopeTokens: item.scopeTokens,
    actionId: item.actionId,
    actionDisplayName: item.actionDisplayName,
    actionLabel: item.actionLabels[0] ?? item.actionId,
    actionDisabled: item.actionDisabled,
    actionUiGenerated: item.actionUiGenerated,
    count: item.count,
  };
}

export function detailFromCrTemplate(
  group: CrTemplateGroupRef,
  template: string,
): UrlTemplateDetail {
  return {
    template,
    scopeTokens: group.scopeTokens,
    actionId: group.actionId,
    actionDisplayName: group.actionDisplayName,
    actionLabel: group.actionLabel,
    actionDisabled: group.actionDisabled,
    actionUiGenerated: group.actionUiGenerated,
    count: 1,
    groupIndex: group.groupIndex,
    groupNotes: group.groupNotes,
  };
}

export function detailFromCrGroup(group: CrTemplateGroupRef, preferredTemplate?: string): UrlTemplateDetail {
  const template =
    preferredTemplate && group.templates.includes(preferredTemplate)
      ? preferredTemplate
      : (group.templates[0] ?? '');
  return detailFromCrTemplate(group, template);
}

export function isActiveRuleGroup(detail: UrlTemplateDetail, group: CrTemplateGroupRef): boolean {
  return (
    detail.actionId === group.actionId &&
    detail.groupIndex === group.groupIndex &&
    scopeTokensSortKey(detail.scopeTokens) === scopeTokensSortKey(group.scopeTokens)
  );
}
