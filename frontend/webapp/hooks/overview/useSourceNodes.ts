import { useState } from 'react';
import { type Node } from '@xyflow/react';
import { nodeConfig, type NodePositions, type EntityCounts } from '@/containers';
import { getMainContainerLanguage } from '@/utils/constants/programming-languages';
import { getEntityIcon, getEntityLabel, getHealthStatus, getProgrammingLanguageIcon } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['k8sActualSources'];
  positions: NodePositions;
  unfilteredCounts: EntityCounts;
  containerHeight: number;
}

const { nodeWidth, nodeHeight, framePadding } = nodeConfig;

const mapToNodeData = (entity: Params['entities'][0]) => {
  return {
    nodeWidth,
    id: {
      namespace: entity.namespace,
      name: entity.name,
      kind: entity.kind,
    },
    type: OVERVIEW_ENTITY_TYPES.SOURCE,
    status: getHealthStatus(entity),
    title: getEntityLabel(entity, OVERVIEW_ENTITY_TYPES.SOURCE, { extended: true }),
    subTitle: entity.kind,
    imageUri: getProgrammingLanguageIcon(getMainContainerLanguage(entity)),
    raw: entity,
  };
};

export const useSourceNodes = ({ entities, positions, unfilteredCounts, containerHeight }: Params) => {
  const [scrollYOffset, setScrollYOffset] = useState(0);

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
        nodeHeight: containerHeight - nodeHeight + framePadding * 2,
        items: entities.map((source) => ({
          id: `source-${source.namespace}-${source.name}-${source.kind}`,
          data: {
            framePadding,
            ...mapToNodeData(source),
          },
        })),
        onScroll: ({ clientHeight, scrollHeight, scrollTop }) => {
          console.log('Node scrolled', { clientHeight, scrollHeight, scrollTop });

          setScrollYOffset(scrollTop);
        },
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
          y: position['y'](idx) - (nodeHeight - framePadding) - scrollYOffset,
        },
        data: mapToNodeData(source),
        style: {
          opacity: 0,
          zIndex: -1,
        },
      });
    });
  }

  return nodes;
};
