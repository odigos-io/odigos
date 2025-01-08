import { type Node } from '@xyflow/react';
import nodeConfig from './node-config.json';
import { type EntityCounts } from './get-entity-counts';
import { type NodePositions } from './get-node-positions';
import { getMainContainerLanguage } from '@/utils/constants/programming-languages';
import { getEntityIcon, getEntityLabel, getHealthStatus, getProgrammingLanguageIcon } from '@/utils';
import { type K8sActualSource, NODE_TYPES, OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES } from '@/types';

interface Params {
  loading: boolean;
  entities: K8sActualSource[];
  positions: NodePositions;
  unfilteredCounts: EntityCounts;
  containerHeight: number;
  onScroll: (params: { clientHeight: number; scrollHeight: number; scrollTop: number }) => void;
}

const { nodeWidth, nodeHeight, framePadding } = nodeConfig;

const mapToNodeData = (entity: Params['entities'][0]) => {
  return {
    nodeWidth,
    nodeHeight,
    framePadding,
    id: {
      namespace: entity.namespace,
      name: entity.name,
      kind: entity.kind,
    },
    type: OVERVIEW_ENTITY_TYPES.SOURCE,
    status: getHealthStatus(entity),
    title: getEntityLabel(entity, OVERVIEW_ENTITY_TYPES.SOURCE, { extended: true }),
    subTitle: `${entity.namespace} • ${entity.kind}`,
    iconSrc: getProgrammingLanguageIcon(getMainContainerLanguage(entity)),
    raw: entity,
  };
};

export const buildSourceNodes = ({ loading, entities, positions, unfilteredCounts, containerHeight, onScroll }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.SOURCE];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.SOURCE];

  nodes.push({
    id: 'source-header',
    type: NODE_TYPES.HEADER,
    position: {
      x: positions[OVERVIEW_ENTITY_TYPES.SOURCE]['x'],
      y: 0,
    },
    data: {
      nodeWidth,
      title: 'Sources',
      icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.SOURCE),
      tagValue: unfilteredCount,
    },
  });

  if (loading) {
    nodes.push({
      id: 'source-skeleton',
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
      id: 'source-add',
      type: NODE_TYPES.ADD,
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        nodeWidth,
        type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
        status: STATUSES.HEALTHY,
        title: 'ADD SOURCE',
        subTitle: 'To collect OpenTelemetry data',
      },
    });
  } else {
    nodes.push({
      id: 'source-scroll',
      type: NODE_TYPES.SCROLL,
      position: {
        x: position['x'],
        y: position['y']() - framePadding,
      },
      data: {
        nodeWidth,
        nodeHeight: containerHeight - nodeHeight + framePadding * 2,
        items: entities.map((source, idx) => ({
          id: `source-${idx}`,
          data: mapToNodeData(source),
        })),
        onScroll,
      },
    });

    entities.forEach((source, idx) => {
      nodes.push({
        id: `source-${idx}-hidden`,
        type: NODE_TYPES.EDGED,
        extent: 'parent',
        parentId: 'source-scroll',
        position: {
          x: framePadding,
          y: position['y'](idx) - (nodeHeight - framePadding),
        },
        style: {
          zIndex: -1,
        },
        data: mapToNodeData(source),
      });
    });
  }

  return nodes;
};
