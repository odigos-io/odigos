'use client';
import React, { useEffect, useMemo } from 'react';
import styled from 'styled-components';
import { buildNodes } from './build-nodes';
import { buildEdges } from './build-edges';
import { NodeDataFlow } from '@/reuseable-components';
import MultiSourceControl from '../multi-source-control';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { useComputePlatform, useContainerSize, useMetrics, useNodeDataFlowHandlers } from '@/hooks';

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

  const nodes = useMemo(() => {
    return buildNodes({
      containerWidth,
      containerHeight,
      computePlatform: data?.computePlatform,
      computePlatformFiltered: filteredData?.computePlatform,
    });
  }, [containerWidth, containerHeight, data, filteredData]);

  const edges = useMemo(() => {
    return buildEdges({
      nodes,
      metrics,
    });
  }, [nodes, metrics]);

  return (
    <Container ref={containerRef}>
      <OverviewActionMenuContainer />
      <MultiSourceControl />
      <NodeDataFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick} />
    </Container>
  );
}
