import { type Node } from '@xyflow/react';
import nodeConfig from './node-config.json';
import { type EntityCounts } from './get-entity-counts';
import { type NodePositions } from './get-node-positions';
import { getActionIcon, getEntityIcon, getEntityLabel } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['actions'];
  positions: NodePositions;
  unfilteredCounts: EntityCounts;
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
    imageUri: getActionIcon(entity.type),
    monitors: entity.spec.signals,
    isActive: !entity.spec.disabled,
    raw: entity,
  };
};

export const buildActionNodes = ({ entities, positions, unfilteredCounts }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.ACTION];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.ACTION];

  nodes.push({
    id: 'action-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.ACTION]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Actions',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.ACTION),
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.ACTION],
    },
  });

  if (!entities.length) {
    nodes.push({
      id: 'action-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_ACTION,
        status: STATUSES.HEALTHY,
        title: 'ADD ACTION',
        subTitle: `Add ${!!unfilteredCount ? 'a new' : 'first'} action to modify the OpenTelemetry data`,
      },
    });
  } else {
    nodes.push({
      id: 'action-frame',
      type: 'frame',
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
        id: `action-${action.id}`,
        type: 'base',
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
