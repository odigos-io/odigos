'use client';
import React, { useMemo } from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { ToastList } from '@/components';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { buildNodesAndEdges, NodeBaseDataFlow } from '@/reuseable-components';
import { useMetrics, useContainerWidth, useNodeDataFlowHandlers, useSourceCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useActionCRUD } from '@/hooks';

const AllDrawers = dynamic(() => import('../all-drawers'), {
  ssr: false,
});

const AllModals = dynamic(() => import('../all-modals'), {
  ssr: false,
});

export const OverviewDataFlowWrapper = styled.div`
  width: calc(100% - 64px);
  height: calc(100vh - 176px);
  position: relative;
`;

const NODE_WIDTH = 255;
const NODE_HEIGHT = 80;

export function OverviewDataFlowContainer() {
  const { metrics } = useMetrics();
  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();
  const { containerRef, containerWidth } = useContainerWidth();
  const { handleNodeClick } = useNodeDataFlowHandlers({
    rules: instrumentationRules,
    sources,
    actions,
    destinations,
  });

  // Memoized node and edge builder to improve performance
  const { nodes, edges } = useMemo(() => {
    return buildNodesAndEdges({
      rules: instrumentationRules,
      sources,
      actions,
      destinations,
      metrics,
      containerWidth,
      nodeWidth: NODE_WIDTH,
      nodeHeight: NODE_HEIGHT,
    });
  }, [instrumentationRules, sources, actions, destinations, metrics, containerWidth]);

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <OverviewActionMenuContainer />
      <NodeBaseDataFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick} nodeWidth={NODE_WIDTH} />

      <AllDrawers />
      <AllModals />
      <ToastList />
    </OverviewDataFlowWrapper>
  );
}
