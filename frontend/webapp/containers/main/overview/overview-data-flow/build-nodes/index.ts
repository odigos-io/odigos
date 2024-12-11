import { type Node } from '@xyflow/react';
import { type ComputePlatformMapped } from '@/types';

import { getCounts } from './get-counts';
import { getPositions } from './get-positions';
import { buildRuleNodes } from './build-rule-nodes';
import { buildSourceNodes } from './build-source-nodes';
import { buildActionNodes } from './build-action-nodes';
import { buildDestinationNodes } from './build-destination-nodes';

interface Params {
  containerWidth: number;
  containerHeight: number;
  computePlatform?: ComputePlatformMapped['computePlatform'];
  computePlatformFiltered?: ComputePlatformMapped['computePlatform'];
}

export const buildNodes = ({ containerWidth, containerHeight, computePlatform, computePlatformFiltered }: Params) => {
  const { instrumentationRules: rules = [], k8sActualSources: sources = [], actions = [], destinations = [] } = computePlatformFiltered || {};
  const nodes: Node[] = [];

  if (!containerWidth) return nodes;

  const positions = getPositions({ containerWidth });
  const unfilteredCounts = getCounts({ computePlatform });

  const ruleNodes = buildRuleNodes({ entities: rules, positions, unfilteredCounts });
  const sourceNodes = buildSourceNodes({ entities: sources, positions, unfilteredCounts, containerHeight });
  const actionNodes = buildActionNodes({ entities: actions, positions, unfilteredCounts });
  const destinationNodes = buildDestinationNodes({ entities: destinations, positions, unfilteredCounts });

  // this is to control the behaviour of the "fit into view" control-button
  nodes.push({
    id: 'hidden',
    type: 'default',
    position: {
      x: containerWidth / 2,
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

  return nodes.concat(ruleNodes, sourceNodes, actionNodes, destinationNodes);
};
