import { type Node } from '@xyflow/react';
import { type Positions } from './get-positions';
import { type UnfilteredCounts } from './get-counts';
import { getMainContainerLanguage } from '@/utils/constants/programming-languages';
import { getEntityIcon, getEntityLabel, getHealthStatus, getProgrammingLanguageIcon } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';
import config from './config.json';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['k8sActualSources'];
  positions: Positions;
  unfilteredCounts: UnfilteredCounts;
  containerHeight: number;
}

const { nodeWidth, nodeHeight, framePadding } = config;

export const buildSourceNodes = ({ entities, positions, unfilteredCounts, containerHeight }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.SOURCE];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.SOURCE];

  nodes.push({
    id: 'source-header',
    type: 'header',
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.SOURCE]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Sources',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.SOURCE),
      tagValue: unfilteredCounts[OVERVIEW_ENTITY_TYPES.SOURCE],
    },
  });

  if (!entities.length) {
    nodes.push({
      id: 'source-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
        status: STATUSES.HEALTHY,
        title: 'ADD SOURCE',
        subTitle: `Add ${!!unfilteredCount ? 'a new' : 'first'} source to collect OpenTelemetry data`,
      },
    });
  } else {
    nodes.push({
      id: 'source-scroll',
      type: 'scroll',
      position: {
        x: position['x'],
        y: position['y']() - framePadding,
      },
      data: {
        nodeWidth,
        nodeHeight: containerHeight - nodeHeight + framePadding,
      },
    });

    entities.forEach((source, idx) => {
      nodes.push({
        id: `source-${source.namespace}-${source.name}-${source.kind}`,
        type: 'base',
        extent: 'parent',
        parentId: 'source-scroll',
        position: {
          x: framePadding,
          y: position['y'](idx) - (nodeHeight - framePadding),
        },
        data: {
          nodeWidth,
          id: {
            namespace: source.namespace,
            name: source.name,
            kind: source.kind,
          },
          type: OVERVIEW_ENTITY_TYPES.SOURCE,
          status: getHealthStatus(source),
          title: getEntityLabel(source, OVERVIEW_ENTITY_TYPES.SOURCE, { extended: true }),
          subTitle: source.kind,
          imageUri: getProgrammingLanguageIcon(getMainContainerLanguage(source)),
          raw: source,
        },
      });
    });
  }

  return nodes;
};
