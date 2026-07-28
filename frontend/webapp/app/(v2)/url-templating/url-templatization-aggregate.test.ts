import { ActionType, type Action } from '@odigos/ui-kit/types';
import {
  aggregateUrlTemplatization,
  deriveUrlTemplatizationPageSummary,
  type ClusterDefaultTemplatizationEffect,
} from './url-templatization-aggregate';

function makeAction(partial: {
  id: string;
  name?: string;
  disabled?: boolean;
  fields?: Action['fields'];
}): Action {
  return {
    id: partial.id,
    type: ActionType.URLTemplatization,
    name: partial.name ?? partial.id,
    disabled: partial.disabled ?? false,
    signals: [],
    fields: partial.fields ?? {},
  } as Action;
}

const enabledClusterWide: ClusterDefaultTemplatizationEffect = {
  mode: 'cluster_wide',
  disabledClusterWide: false,
  title: '',
  description: '',
  sectionSummary: '',
};

const disabledClusterWide: ClusterDefaultTemplatizationEffect = {
  mode: 'cluster_wide',
  disabledClusterWide: true,
  title: '',
  description: '',
  sectionSummary: '',
};

const scoped: ClusterDefaultTemplatizationEffect = {
  mode: 'scoped',
  disabledClusterWide: false,
  title: '',
  description: '',
  sectionSummary: '',
};

const none: ClusterDefaultTemplatizationEffect = {
  mode: 'none',
  disabledClusterWide: false,
  title: '',
  description: '',
  sectionSummary: '',
};

describe('deriveUrlTemplatizationPageSummary', () => {
  it('reports OFF when there are no enabled actions', () => {
    expect(deriveUrlTemplatizationPageSummary(enabledClusterWide, 0, 0)).toBe('URL templating is OFF');
  });

  it('reports OFF when default is disabled cluster-wide with no custom rules', () => {
    expect(deriveUrlTemplatizationPageSummary(disabledClusterWide, 0, 1)).toBe('URL templating is OFF');
  });

  it('reports default OFF with custom rules when cluster-wide default is disabled', () => {
    expect(deriveUrlTemplatizationPageSummary(disabledClusterWide, 3, 1)).toBe(
      'Default templating is OFF · 3 custom rules',
    );
  });

  it('reports entire-cluster default when explicitly enabled', () => {
    expect(deriveUrlTemplatizationPageSummary(enabledClusterWide, 0, 1)).toBe(
      'Applying default templating to the entire cluster',
    );
  });

  it('reports default OFF when no default rules are configured', () => {
    expect(deriveUrlTemplatizationPageSummary(none, 0, 1)).toBe('URL templating is OFF');
    expect(deriveUrlTemplatizationPageSummary(none, 2, 1)).toBe(
      'Default templating is OFF · 2 custom rules',
    );
  });

  it('reports scoped default with optional custom rules', () => {
    expect(deriveUrlTemplatizationPageSummary(scoped, 0, 1)).toBe(
      'Applying default templating to specific scopes',
    );
    expect(deriveUrlTemplatizationPageSummary(scoped, 1, 1)).toBe(
      'Applying default templating to specific scopes · 1 custom rule',
    );
  });
});

describe('aggregateUrlTemplatization pageSummary', () => {
  it('ignores disabled actions when computing the page summary', () => {
    const aggregation = aggregateUrlTemplatization([
      makeAction({
        id: 'disabled',
        disabled: true,
        fields: {
          urlTemplatizationRulesGroups: [{ templatizationRules: [{ template: '/users/{id}' }] }],
        },
      }),
    ]);
    expect(aggregation.enabledActionCount).toBe(0);
    expect(aggregation.enabledCustomRuleInstances).toBe(0);
    expect(aggregation.pageSummary).toBe('URL templating is OFF');
  });

  it('summarizes cluster-wide default with custom rules', () => {
    const aggregation = aggregateUrlTemplatization([
      makeAction({
        id: 'global',
        fields: {
          urlTemplatizationDefaultGroups: [{}],
          urlTemplatizationRulesGroups: [
            { templatizationRules: [{ template: '/users/{id}' }, { template: '/orders/{id}' }] },
          ],
        },
      }),
    ]);
    expect(aggregation.pageSummary).toBe(
      'Applying default templating to the entire cluster · 2 custom rules',
    );
  });
});
