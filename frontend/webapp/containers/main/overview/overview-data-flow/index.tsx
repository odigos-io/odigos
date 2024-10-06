'use client';
import styled from 'styled-components';
import { OverviewDrawer } from '../overview-drawer';
import React, { useMemo, useRef, useEffect, useState } from 'react';
import { OverviewActionMenuContainer } from '../overview-actions-menu';
import { buildNodesAndEdges, NodeBaseDataFlow } from '@/reuseable-components';
import { useActualDestination, useActualSources, useGetActions } from '@/hooks';

export const OverviewDataFlowWrapper = styled.div`
  width: calc(100% - 64px);
  height: calc(100vh - 176px);
  position: relative;
`;

export function OverviewDataFlowContainer() {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [containerWidth, setContainerWidth] = useState<number>(0);

  const { actions } = useGetActions();
  const { sources } = useActualSources();
  const { destinations } = useActualDestination();

  // Get the width of the container dynamically
  useEffect(() => {
    if (containerRef.current) {
      setContainerWidth(
        containerRef.current.getBoundingClientRect().width - 64
      );
    }

    const handleResize = () => {
      if (containerRef.current) {
        setContainerWidth(
          containerRef.current.getBoundingClientRect().width - 64
        );
      }
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const columnWidth = 296;

  // Memoized node and edge builder to improve performance
  const { nodes, edges } = useMemo(() => {
    return buildNodesAndEdges({
      sources,
      actions,
      destinations,
      columnWidth,
      containerWidth,
    });
  }, [sources, actions, destinations, columnWidth, containerWidth]);

  function onNodeClick(_, object: any) {
    console.log(object);
  }

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <OverviewDrawer />
      <OverviewActionMenuContainer />
      <NodeBaseDataFlow nodes={nodes} edges={edges} onNodeClick={onNodeClick} />
    </OverviewDataFlowWrapper>
  );
}
