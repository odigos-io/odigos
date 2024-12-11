import theme from '@/styles/theme';
import { type Node } from '@xyflow/react';
import { getActionIcon, getEntityLabel } from '@/utils';
import { Positions, UnfilteredCounts } from './build-layout-nodes';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES, type ComputePlatformMapped } from '@/types';
import { nodeWidth, nodeHeight } from './config.json';

interface Params {
  entities: ComputePlatformMapped['computePlatform']['actions'];
  positions: Positions;
  unfilteredCounts: UnfilteredCounts;
}

export const buildActionNodes = ({ entities, positions, unfilteredCounts }: Params) => {
  const nodes: Node[] = [];
  const position = positions[OVERVIEW_ENTITY_TYPES.ACTION];
  const unfilteredCount = unfilteredCounts[OVERVIEW_ENTITY_TYPES.ACTION];

  if (!entities.length) {
    nodes.push({
      id: 'action-add',
      type: 'add',
      position: {
        x: position['x'],
        y: position['y'](),
      },
      data: {
        type: OVERVIEW_NODE_TYPES.ADD_ACTION,
        status: STATUSES.HEALTHY,
        title: 'ADD ACTION',
        subTitle: `Add ${!!unfilteredCount ? 'a new' : 'first'} action to modify the OpenTelemetry data`,
      },
    });
  } else {
    const groupPadding = 12;

    nodes.push({
      id: 'action-group',
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
        border: `1px dashed ${theme.colors.border}`,
        borderRadius: 24,
      },
    });

    entities.forEach((action, idx) => {
      nodes.push({
        id: `action-${action.id}`,
        type: 'base',
        extent: 'parent',
        parentId: 'action-group',
        position: {
          x: groupPadding,
          y: position['y'](idx) - (nodeHeight - groupPadding),
        },
        data: {
          id: action.id,
          type: OVERVIEW_ENTITY_TYPES.ACTION,
          status: STATUSES.HEALTHY,
          title: getEntityLabel(action, OVERVIEW_ENTITY_TYPES.ACTION, { prioritizeDisplayName: true }),
          subTitle: action.type,
          imageUri: getActionIcon(action.type),
          monitors: action.spec.signals,
          isActive: !action.spec.disabled,
          raw: action,
        },
      });
    });
  }

  return nodes;
};
