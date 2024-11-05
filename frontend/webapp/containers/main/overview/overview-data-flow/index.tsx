'use client';
import React, { useMemo } from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { buildNodesAndEdges, NodeBaseDataFlow } from '@/reuseable-components';
import {
  useMetrics,
  useGetActions,
  useActualSources,
  useContainerWidth,
  useActualDestination,
  useNodeDataFlowHandlers,
  useGetInstrumentationRules,
} from '@/hooks';

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
  const { actions } = useGetActions();
  const { sources } = useActualSources();
  const { destinations } = useActualDestination();
  const { instrumentationRules } = useGetInstrumentationRules();
  const { containerRef, containerWidth } = useContainerWidth();
  const { handleNodeClick } = useNodeDataFlowHandlers({
    rules: instrumentationRules,
    sources,
    actions,
    destinations,
  });

  // TODO: remove mockup data from "useMetrics"
  const { metrics } = useMetrics({ sources, destinations });

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
    </OverviewDataFlowWrapper>
  );
}
