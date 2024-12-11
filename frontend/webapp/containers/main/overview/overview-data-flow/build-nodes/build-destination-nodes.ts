import { type Node } from '@xyflow/react';
import { type Positions } from './get-positions';
import { type UnfilteredCounts } from './get-counts';
import { extractMonitors, getEntityIcon, getEntityLabel, getHealthStatus } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';
import config from './config.json';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['destinations'];
  positions: Positions;
  unfilteredCounts: UnfilteredCounts;
}

const { nodeWidth } = config;

export const buildDestinationNodes = ({ entities, positions, unfilteredCounts }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.DESTINATION];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.DESTINATION];

  nodes.push({
    id: 'destination-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.DESTINATION]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Destinations',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.DESTINATION),
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.DESTINATION],
    },
  });

  if (!entities.length) {
    nodes.push({
      id: 'destination-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_DESTIONATION,
        status: STATUSES.HEALTHY,
        title: 'ADD DESTIONATION',
        subTitle: `Add ${!!unfilteredCount ? 'a new' : 'first'} destination to monitor the OpenTelemetry data`,
      },
    });
  } else {
    entities.forEach((destination, idx) => {
      nodes.push({
        id: `destination-${destination.id}`,
        type: 'base',
        position: {
          x: position['x'],
          y: position['y'](idx),
        },
        data: {
          nodeWidth,
          id: destination.id,
          type: OVERVIEW_ENTITY_TYPES.DESTINATION,
          status: getHealthStatus(destination),
          title: getEntityLabel(destination, OVERVIEW_ENTITY_TYPES.DESTINATION, { prioritizeDisplayName: true }),
          subTitle: destination.destinationType.displayName,
          imageUri: destination.destinationType.imageUrl || '/brand/odigos-icon.svg',
          monitors: extractMonitors(destination.exportedSignals),
          raw: destination,
        },
      });
    });
  }

  return nodes;
};
