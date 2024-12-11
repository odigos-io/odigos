import { type Node } from '@xyflow/react';
import { getEntityIcon, getValueForRange } from '@/utils';
import { type ComputePlatformMapped, OVERVIEW_ENTITY_TYPES } from '@/types';
import { nodeWidth, nodeHeight } from './config.json';

interface Params {
  containerWidth: number;
  containerHeight: number;
  // must be non-filtered
  computePlatform?: ComputePlatformMapped['computePlatform'];
}

export type Positions = Record<
  OVERVIEW_ENTITY_TYPES,
  {
    x: number;
    y: (idx?: number) => number;
  }
>;

export type UnfilteredCounts = Record<OVERVIEW_ENTITY_TYPES, number>;

export const buildLayoutNodes = ({ containerWidth, containerHeight, computePlatform }: Params) => {
  const nodes: Node[] = [];

  const startX = 24;
  const endX = (containerWidth <= 1500 ? 1500 : containerWidth) - nodeWidth - 40 - startX;
  const getY = (idx?: number) => nodeHeight * ((idx || 0) + 1);

  const positions: Positions = {
    [OVERVIEW_ENTITY_TYPES.RULE]: {
      x: startX,
      y: getY,
    },
    [OVERVIEW_ENTITY_TYPES.SOURCE]: {
      x: getValueForRange(containerWidth, [
        [0, 1600, endX / 3.5],
        [1600, null, endX / 4],
      ]),
      y: getY,
    },
    [OVERVIEW_ENTITY_TYPES.ACTION]: {
      x: getValueForRange(containerWidth, [
        [0, 1600, endX / 1.55],
        [1600, null, endX / 1.6],
      ]),
      y: getY,
    },
    [OVERVIEW_ENTITY_TYPES.DESTINATION]: {
      x: endX,
      y: getY,
    },
  };

  const unfilteredCounts: UnfilteredCounts = {
    [OVERVIEW_ENTITY_TYPES.RULE]: computePlatform?.instrumentationRules.length || 0,
    [OVERVIEW_ENTITY_TYPES.SOURCE]: computePlatform?.k8sActualSources.length || 0,
    [OVERVIEW_ENTITY_TYPES.ACTION]: computePlatform?.actions.length || 0,
    [OVERVIEW_ENTITY_TYPES.DESTINATION]: computePlatform?.destinations.length || 0,
  };

  if (!containerWidth) return { nodes, positions, unfilteredCounts };

  nodes.push({
    id: 'rule-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.RULE]['x'],
      y: 0,
    },
    data: {
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.RULE),
      title: 'Instrumentation Rules',
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.RULE],
    },
  });

  nodes.push({
    id: 'source-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.SOURCE]['x'],
      y: 0,
    },
    data: {
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.SOURCE),
      title: 'Sources',
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.SOURCE],
    },
  });

  nodes.push({
    id: 'action-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.ACTION]['x'],
      y: 0,
    },
    data: {
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.ACTION),
      title: 'Actions',
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.ACTION],
    },
  });

  nodes.push({
    id: 'destination-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.DESTINATION]['x'],
      y: 0,
    },
    data: {
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.DESTINATION),
      title: 'Destinations',
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.DESTINATION],
    },
  });

  // this is to control the behaviour of the "fit into view" control-button
  nodes.push({
    id: 'hidden',
    type: 'default',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.RULE]['x'],
      y: containerHeight,
    },
    data: {},
    style: {
      width: 1,
      height: 1,
      opacity: 0,
      pointerEvents: 'none',
    },
  });

  return { nodes, positions, unfilteredCounts };
};
