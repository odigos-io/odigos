import { type Node } from '@xyflow/react';
import nodeConfig from './node-config.json';
import { type EntityCounts } from './get-entity-counts';
import { type NodePositions } from './get-node-positions';
import { getEntityIcon, getEntityLabel, getRuleIcon } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['instrumentationRules'];
  positions: NodePositions;
  unfilteredCounts: EntityCounts;
}

const { nodeWidth } = nodeConfig;

const mapToNodeData = (entity: Params['entities'][0]) => {
  return {
    nodeWidth,
    id: entity.ruleId,
    type: OVERVIEW_ENTITY_TYPES.RULE,
    status: STATUSES.HEALTHY,
    title: getEntityLabel(entity, OVERVIEW_ENTITY_TYPES.RULE, { prioritizeDisplayName: true }),
    subTitle: entity.type,
    imageUri: getRuleIcon(entity.type),
    isActive: !entity.disabled,
    raw: entity,
  };
};

export const buildRuleNodes = ({ entities, positions, unfilteredCounts }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.RULE];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.RULE];

  nodes.push({
    id: 'rule-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.RULE]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Instrumentation Rules',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.RULE),
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.RULE],
    },
  });

  if (!entities.length) {
    nodes.push({
      id: 'rule-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
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
        data: mapToNodeData(rule),
      });
    });
  }

  return nodes;
};
