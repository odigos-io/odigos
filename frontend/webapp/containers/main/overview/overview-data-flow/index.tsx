'use client';
import React, { useEffect, useMemo } from 'react';
import styled from 'styled-components';
import MultiSourceControl from '../multi-source-control';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { buildNodesAndEdges, NodeBaseDataFlow } from '@/reuseable-components';
import { useComputePlatform, useContainerSize, useMetrics, useNodeDataFlowHandlers } from '@/hooks';

const OverviewDataFlowWrapper = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const NODE_WIDTH = 255;
const NODE_HEIGHT = 80;

export default function OverviewDataFlowContainer() {
  const { containerRef, containerWidth, containerHeight } = useContainerSize();
  const { data, filteredData, startPolling } = useComputePlatform();
  const { handleNodeClick } = useNodeDataFlowHandlers();
  const { metrics } = useMetrics();

  useEffect(() => {
    // this is to start polling on component mount in an attempt to fix any initial errors with sources/destinations
    if (!!data?.computePlatform.k8sActualSources.length || !!data?.computePlatform.destinations.length) startPolling();
    // only on-mount, if we include "data" this might trigger on every refetch
  }, []);

  // Memoized node and edge builder to improve performance
  const { nodes, edges } = useMemo(() => {
    return buildNodesAndEdges({
      computePlatform: data?.computePlatform,
      computePlatformFiltered: filteredData?.computePlatform,
      metrics,
      containerWidth,
      containerHeight,
      nodeWidth: NODE_WIDTH,
      nodeHeight: NODE_HEIGHT,
    });
  }, [data, filteredData, metrics, containerWidth, containerHeight]);

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <OverviewActionMenuContainer />
      <MultiSourceControl />
      <NodeBaseDataFlow nodes={nodes} edges={edges} nodeWidth={NODE_WIDTH} onNodeClick={handleNodeClick} />
    </OverviewDataFlowWrapper>
  );
}
