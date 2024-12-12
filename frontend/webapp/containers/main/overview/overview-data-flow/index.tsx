'use client';
import React, { useEffect, useMemo } from 'react';
import { type Node } from '@xyflow/react';
import styled from 'styled-components';
import nodeConfig from './node-config.json';
import { buildEdges } from './build-edges';
import { getEntityCounts } from './get-entity-counts';
import { getNodePositions } from './get-node-positions';
import { NodeDataFlow } from '@/reuseable-components';
import MultiSourceControl from '../multi-source-control';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { useActionNodes, useComputePlatform, useContainerSize, useDestinationNodes, useMetrics, useNodeDataFlowHandlers, useRuleNodes, useSourceNodes } from '@/hooks';

export * from './get-entity-counts';
export * from './get-node-positions';
export { nodeConfig };

const Container = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

export default function OverviewDataFlowContainer() {
  const { containerRef, containerWidth, containerHeight } = useContainerSize();
  const { handleNodeClick } = useNodeDataFlowHandlers();

  const { data, filteredData, startPolling } = useComputePlatform();
  const { metrics } = useMetrics();

  useEffect(() => {
    // this is to start polling on component mount in an attempt to fix any initial errors with sources/destinations
    if (!!data?.computePlatform.k8sActualSources.length || !!data?.computePlatform.destinations.length) startPolling();
    // only on-mount, if we include "data" this might trigger on every refetch
  }, []);

  const positions = useMemo(() => getNodePositions({ containerWidth }), [containerWidth]);
  const unfilteredCounts = useMemo(() => getEntityCounts({ computePlatform: data?.computePlatform }), [data]);

  const actionNodes = useActionNodes({ entities: filteredData?.computePlatform.actions || [], positions, unfilteredCounts });
  const ruleNodes = useRuleNodes({ entities: filteredData?.computePlatform.instrumentationRules || [], positions, unfilteredCounts });
  const destinationNodes = useDestinationNodes({ entities: filteredData?.computePlatform.destinations || [], positions, unfilteredCounts });
  const sourceNodes = useSourceNodes({ entities: filteredData?.computePlatform.k8sActualSources || [], positions, unfilteredCounts, containerHeight });

  const nodes = useMemo(() => ([] as Node[]).concat(actionNodes, ruleNodes, sourceNodes, destinationNodes), [actionNodes, ruleNodes, sourceNodes, destinationNodes]);
  const edges = useMemo(() => buildEdges({ nodes, metrics }), [nodes, metrics]);

  return (
    <Container ref={containerRef}>
      <OverviewActionMenuContainer />
      <MultiSourceControl />
      <NodeDataFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick} />
    </Container>
  );
}
