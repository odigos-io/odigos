'use client';
import React, { useMemo } from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { ToastList } from '@/components';
import MultiSourceControl from '../multi-source-control';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { buildNodesAndEdges, NodeBaseDataFlow } from '@/reuseable-components';
import { useComputePlatform, useContainerSize, useMetrics, useNodeDataFlowHandlers } from '@/hooks';

const AllDrawers = dynamic(() => import('../all-drawers'), { ssr: false });
const AllModals = dynamic(() => import('../all-modals'), { ssr: false });

export const OverviewDataFlowWrapper = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const NODE_WIDTH = 255;
const NODE_HEIGHT = 80;

export default function OverviewDataFlowContainer() {
  const { containerRef, containerWidth, containerHeight } = useContainerSize();
  const { handleNodeClick } = useNodeDataFlowHandlers();
  const { data, filteredData } = useComputePlatform();
  const { metrics } = useMetrics();

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

      <AllDrawers />
      <AllModals />
      <ToastList />
    </OverviewDataFlowWrapper>
  );
}
