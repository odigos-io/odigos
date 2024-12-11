import { type Node } from '@xyflow/react';
import { Positions, UnfilteredCounts } from './build-layout-nodes';
import { getMainContainerLanguage } from '@/utils/constants/programming-languages';
import { getEntityLabel, getHealthStatus, getProgrammingLanguageIcon } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';
import { nodeWidth, nodeHeight } from './config.json';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['k8sActualSources'];
  positions: Positions;
  unfilteredCounts: UnfilteredCounts;
}

export const buildSourceNodes = ({ entities, positions, unfilteredCounts }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.SOURCE];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.SOURCE];

  if (!entities.length) {
    nodes.push({
      id: 'source-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
        status: STATUSES.HEALTHY,
        title: 'ADD SOURCE',
        subTitle: `Add ${!!unfilteredCount ? 'a new' : 'first'} source to collect OpenTelemetry data`,
      },
    });
  } else {
    const groupPadding = 12;

    nodes.push({
      id: 'source-group',
      type: 'group',
      position: {
        x: position['x'] - groupPadding,
        y: position['y']() - groupPadding,
      },
      data: {},
      style: {
        width: nodeWidth + groupPadding + 50,
        height: nodeHeight * entities.length + groupPadding,
        background: 'transparent',
        border: 'none',
      },
    });

    entities.forEach((source, idx) => {
      nodes.push({
        id: `source-${source.namespace}-${source.name}-${source.kind}`,
        type: 'base',
        extent: 'parent',
        parentId: 'source-group',
        position: {
          x: groupPadding,
          y: position['y'](idx) - (nodeHeight - groupPadding),
        },
        data: {
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
