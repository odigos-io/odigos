import { type Node } from '@xyflow/react';
import { extractMonitors } from '@/utils';
import nodeConfig from './node-config.json';
import { type NodePositions } from './get-node-positions';
import { type ActualDestination, NODE_TYPES, OVERVIEW_NODE_TYPES } from '@/types';
import { ENTITY_TYPES, getEntityIcon, getEntityLabel, getHealthStatus, HEALTH_STATUS } from '@odigos/ui-utils';

interface Params {
  loading: boolean;
  entities: ActualDestination[];
  positions: NodePositions;
  unfilteredCount: number;
}

const { nodeWidth } = nodeConfig;

const mapToNodeData = (entity: Params['entities'][0]) => {
  return {
    nodeWidth,
    id: entity.id,
    type: ENTITY_TYPES.DESTINATION,
    status: getHealthStatus(entity),
    title: getEntityLabel(entity, ENTITY_TYPES.DESTINATION, { prioritizeDisplayName: true }),
    subTitle: entity.destinationType.displayName,
    iconSrc: entity.destinationType.imageUrl,
    monitors: extractMonitors(entity.exportedSignals),
    raw: entity,
  };
};

export const buildDestinationNodes = ({ loading, entities, positions, unfilteredCount }: Params) => {
  const nodes: Node[] = [];
  const position = positions[ENTITY_TYPES.DESTINATION];

  nodes.push({
    id: 'destination-header',
    type: NODE_TYPES.HEADER,
    position: {
      x: positions[ENTITY_TYPES.DESTINATION]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Destinations',
      icon: getEntityIcon(ENTITY_TYPES.DESTINATION),
      tagValue: unfilteredCount,
    },
  });

  if (loading) {
    nodes.push({
      id: 'destination-skeleton',
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
      id: 'destination-add',
      type: NODE_TYPES.ADD,
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_DESTINATION,
        status: HEALTH_STATUS.HEALTHY,
        title: 'ADD DESTINATION',
        subTitle: 'To monitor OpenTelemetry data',
      },
    });
  } else {
    entities.forEach((destination, idx) => {
      nodes.push({
        id: `destination-${idx}`,
        type: NODE_TYPES.BASE,
        position: {
          x: position['x'],
          y: position['y'](idx),
        },
        data: mapToNodeData(destination),
      });
    });
  }

  return nodes;
};
