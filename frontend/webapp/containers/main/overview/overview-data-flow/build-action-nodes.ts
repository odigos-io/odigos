import { type Node } from '@xyflow/react';
import nodeConfig from './node-config.json';
import { type NodePositions } from './get-node-positions';
import { getActionIcon, getEntityIcon, getEntityLabel } from '@/utils';
import { type ActionDataParsed, NODE_TYPES, OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES } from '@/types';

interface Params {
  loading: boolean;
  entities: ActionDataParsed[];
  positions: NodePositions;
  unfilteredCount: number;
}

const { nodeWidth, nodeHeight, framePadding } = nodeConfig;

const mapToNodeData = (entity: Params['entities'][0]) => {
  return {
    nodeWidth,
    id: entity.id,
    type: OVERVIEW_ENTITY_TYPES.ACTION,
    status: STATUSES.HEALTHY,
    title: getEntityLabel(entity, OVERVIEW_ENTITY_TYPES.ACTION, { prioritizeDisplayName: true }),
    subTitle: entity.type,
    icon: getActionIcon(entity.type),
    monitors: entity.spec.signals,
    isActive: !entity.spec.disabled,
    raw: entity,
  };
};

export const buildActionNodes = ({ loading, entities, positions, unfilteredCount }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.ACTION];

  nodes.push({
    id: 'action-header',
    type: NODE_TYPES.HEADER,
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.ACTION]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Actions',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.ACTION),
      tagValue: unfilteredCount,
    },
  });

  if (loading) {
    nodes.push({
      id: 'action-skeleton',
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
      id: 'action-add',
      type: NODE_TYPES.ADD,
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_ACTION,
        status: STATUSES.HEALTHY,
        title: 'ADD ACTION',
        subTitle: 'To modify OpenTelemetry data',
      },
    });
  } else {
    nodes.push({
      id: 'action-frame',
      type: NODE_TYPES.FRAME,
      position: {
        x: position['x'] - framePadding,
        y: position['y']() - framePadding,
      },
      data: {
        nodeWidth: nodeWidth + 2 * framePadding,
        nodeHeight: nodeHeight * entities.length + framePadding,
      },
    });

    entities.forEach((action, idx) => {
      nodes.push({
        id: `action-${idx}`,
        type: NODE_TYPES.BASE,
        extent: 'parent',
        parentId: 'action-frame',
        position: {
          x: framePadding,
          y: position['y'](idx) - (nodeHeight - framePadding),
        },
        data: mapToNodeData(action),
      });
    });
  }

  return nodes;
};
