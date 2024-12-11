import { type Node } from '@xyflow/react';
import { type ComputePlatformMapped } from '@/types';
import { buildLayoutNodes } from './build-layout-nodes';
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
  const nodes: Node[] = [];

  if (!containerWidth) return nodes;

  const { instrumentationRules: rules = [], k8sActualSources: sources = [], actions = [], destinations = [] } = computePlatformFiltered || {};

  const { nodes: layoutNodes, positions, unfilteredCounts } = buildLayoutNodes({ containerWidth, containerHeight, computePlatform });
  const ruleNodes = !!layoutNodes.length ? buildRuleNodes({ entities: rules, positions, unfilteredCounts }) : [];
  const sourceNodes = !!layoutNodes.length ? buildSourceNodes({ entities: sources, positions, unfilteredCounts }) : [];
  const actionNodes = !!layoutNodes.length ? buildActionNodes({ entities: actions, positions, unfilteredCounts }) : [];
  const destinationNodes = !!layoutNodes.length ? buildDestinationNodes({ entities: destinations, positions, unfilteredCounts }) : [];

  return layoutNodes.concat(ruleNodes, sourceNodes, actionNodes, destinationNodes);
};
