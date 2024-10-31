'use client';
import React, { useMemo } from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { buildNodesAndEdges, NodeBaseDataFlow } from '@/reuseable-components';
import { useGetActions, useActualSources, useContainerWidth, useActualDestination, useNodeDataFlowHandlers } from '@/hooks';
import { useGetInstrumentationRules } from '@/hooks/instrumentation-rules/useGetInstrumentationRules';

const OverviewDrawer = dynamic(() => import('../overview-drawer'), {
  ssr: false,
});

export const OverviewDataFlowWrapper = styled.div`
  width: calc(100% - 64px);
  height: calc(100vh - 176px);
  position: relative;
`;

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

  const columnWidth = 255;

  // Memoized node and edge builder to improve performance
  const { nodes, edges } = useMemo(() => {
    return buildNodesAndEdges({
      rules: instrumentationRules,
      sources,
      actions,
      destinations,
      columnWidth,
      containerWidth,
    });
  }, [instrumentationRules, sources, actions, destinations, columnWidth, containerWidth]);

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <OverviewDrawer />
      <OverviewActionMenuContainer />
      <NodeBaseDataFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick} columnWidth={columnWidth} />
    </OverviewDataFlowWrapper>
  );
}
