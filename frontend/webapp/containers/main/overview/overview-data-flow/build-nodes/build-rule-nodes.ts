import { type Node } from '@xyflow/react';
import { getEntityLabel, getRuleIcon } from '@/utils';
import { Positions, UnfilteredCounts } from './build-layout-nodes';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['instrumentationRules'];
  positions: Positions;
  unfilteredCounts: UnfilteredCounts;
}

export const buildRuleNodes = ({ entities, positions, unfilteredCounts }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.RULE];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.RULE];

  if (!entities.length) {
    nodes.push({
      id: 'rule-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        type: OVERVIEW_NODE_TYPES.ADD_RULE,
        status: STATUSES.HEALTHY,
        title: 'ADD RULE',
        subTitle: `Add ${!!unfilteredCount ? 'a new' : 'first'} rule to modify the OpenTelemetry data`,
      },
    });
  } else {
    entities.forEach((rule, idx) => {
      nodes.push({
        id: `rule-${rule.ruleId}`,
        type: 'base',
        position: {
          x: position['x'],
          y: position['y'](idx),
        },
        data: {
          id: rule.ruleId,
          type: OVERVIEW_ENTITY_TYPES.RULE,
          status: STATUSES.HEALTHY,
          title: getEntityLabel(rule, OVERVIEW_ENTITY_TYPES.RULE, { prioritizeDisplayName: true }),
          subTitle: rule.type,
          imageUri: getRuleIcon(rule.type),
          isActive: !rule.disabled,
          raw: rule,
        },
      });
    });
  }

  return nodes;
};
