import { type Node } from '@xyflow/react';
import nodeConfig from './node-config.json';
import { type NodePositions } from './get-node-positions';
import { extractMonitors, getEntityIcon, getEntityLabel, getHealthStatus } from '@/utils';
import { type ActualDestination, NODE_TYPES, OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES } from '@/types';

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
    type: OVERVIEW_ENTITY_TYPES.DESTINATION,
    status: getHealthStatus(entity),
    title: getEntityLabel(entity, OVERVIEW_ENTITY_TYPES.DESTINATION, { prioritizeDisplayName: true }),
    subTitle: entity.destinationType.displayName,
    iconSrc: entity.destinationType.imageUrl,
    monitors: extractMonitors(entity.exportedSignals),
    raw: entity,
  };
};

export const buildDestinationNodes = ({ loading, entities, positions, unfilteredCount }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.DESTINATION];

  nodes.push({
    id: 'destination-header',
    type: NODE_TYPES.HEADER,
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.DESTINATION]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Destinations',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.DESTINATION),
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
        status: STATUSES.HEALTHY,
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
