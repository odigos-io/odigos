import { type Node } from '@xyflow/react';
import nodeConfig from './node-config.json';
import { type NodePositions } from './get-node-positions';
import { type InstrumentationRuleSpecMapped, NODE_TYPES, OVERVIEW_NODE_TYPES } from '@/types';
import { ENTITY_TYPES, getEntityIcon, getEntityLabel, getInstrumentationRuleIcon, HEALTH_STATUS } from '@odigos/ui-utils';

interface Params {
  loading: boolean;
  entities: InstrumentationRuleSpecMapped[];
  positions: NodePositions;
  unfilteredCount: number;
}

const { nodeWidth } = nodeConfig;

const mapToNodeData = (entity: Params['entities'][0]) => {
  return {
    nodeWidth,
    id: entity.ruleId,
    type: ENTITY_TYPES.INSTRUMENTATION_RULE,
    status: HEALTH_STATUS.HEALTHY,
    title: getEntityLabel(entity, ENTITY_TYPES.INSTRUMENTATION_RULE, { prioritizeDisplayName: true }),
    subTitle: entity.type,
    icon: getInstrumentationRuleIcon(entity.type),
    isActive: !entity.disabled,
    raw: entity,
  };
};

export const buildRuleNodes = ({ loading, entities, positions, unfilteredCount }: Params) => {
  const nodes: Node[] = [];
  const position = positions[ENTITY_TYPES.INSTRUMENTATION_RULE];

  nodes.push({
    id: 'rule-header',
    type: NODE_TYPES.HEADER,
    position: {
      x: positions[ENTITY_TYPES.INSTRUMENTATION_RULE]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Instrumentation Rules',
      icon: getEntityIcon(ENTITY_TYPES.INSTRUMENTATION_RULE),
      tagValue: unfilteredCount,
    },
  });

  if (loading) {
    nodes.push({
      id: 'rule-skeleton',
      type: NODE_TYPES.SKELETON,
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        size: 3,
      },
    });
  } else if (!entities.length) {
    nodes.push({
      id: 'rule-add',
      type: NODE_TYPES.ADD,
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_RULE,
        status: HEALTH_STATUS.HEALTHY,
        title: 'ADD RULE',
        subTitle: 'To modify OpenTelemetry data',
      },
    });
  } else {
    entities.forEach((rule, idx) => {
      nodes.push({
        id: `rule-${idx}`,
        type: NODE_TYPES.BASE,
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
