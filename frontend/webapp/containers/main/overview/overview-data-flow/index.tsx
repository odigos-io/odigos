'use client';
import styled from 'styled-components';
import { NodeBaseDataFlow } from './graph';
import { buildNodesAndEdges } from './graph/builder';
import React, { useMemo, useRef, useEffect, useState } from 'react';
import { useActualDestination, useActualSources, useGetActions } from '@/hooks';

export const OverviewDataFlowWrapper = styled.div`
  width: calc(100% - 64px);
  height: calc(100vh - 100px);
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

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <NodeBaseDataFlow nodes={nodes} edges={edges} />
    </OverviewDataFlowWrapper>
  );
}
