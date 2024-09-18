'use client';
import React, { useMemo, useRef, useEffect, useState } from 'react';
import styled from 'styled-components';
import { NodeBaseDataFlow } from './graph';
import { useActualDestination, useActualSources, useGetActions } from '@/hooks';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import { ReactFlowProvider } from '@xyflow/react';

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

  const columnWidth = 296; // Assuming each column's width is 296px
  const leftColumnX = 0;
  const rightColumnX = containerWidth - columnWidth;
  const centerColumnX = (containerWidth - columnWidth) / 2;

  // Source Nodes
  const sourcesNode = useMemo(() => {
    const headerSource = {
      type: 'header',
      id: 'header-source',
      position: { x: leftColumnX, y: 0 },
      data: {
        icon: '/icons/overview/sources.svg',
        title: 'Sources',
        tagValue: sources.length,
      },
    };

    return [
      headerSource,
      ...sources.map((source, index) => ({
        type: 'base',
        id: `source-${index}`,
        position: { x: leftColumnX, y: 80 * (index + 1) },
        data: {
          type: 'source',
          title: source.name,
          subTitle: source.kind,
          imageUri: getMainContainerLanguageLogo(source),
          status: 'healthy',
          onClick: () => {
            console.log(source);
          },
        },
      })),
    ];
  }, [sources, leftColumnX]);

  // Destination Nodes
  const destinationNode = useMemo(() => {
    const headerDestination = {
      type: 'header',
      id: 'header-destination',
      position: { x: rightColumnX, y: 0 },
      data: {
        icon: '/icons/overview/destinations.svg',
        title: 'Destinations',
        tagValue: destinations.length,
      },
    };

    return [
      headerDestination,
      ...destinations.map((destination, index) => ({
        type: 'base',
        id: `destination-${index}`,
        position: { x: rightColumnX, y: 80 * (index + 1) },
        data: {
          type: 'destination',
          title: destination.destinationType.displayName,
          subTitle: 'Destination',
          imageUri: destination.destinationType.imageUrl,
          monitors: destination.exportedSignals,
          status: 'healthy',
          onClick: () => {
            console.log(destination);
          },
        },
      })),
    ];
  }, [destinations, rightColumnX]);

  // Actions Nodes
  const actionsNode = useMemo(() => {
    const headerAction = {
      type: 'header',
      id: 'header-action',
      position: { x: centerColumnX, y: 0 },
      data: {
        icon: '/icons/overview/actions.svg',
        title: 'Actions',
        tagValue: actions.length,
      },
    };

    return [
      headerAction,
      ...actions.map((action, index) => ({
        type: 'base',
        id: `action-${index}`,
        position: { x: centerColumnX, y: 80 * (index + 1) },
        data: {
          type: 'action',
          title: action.type,
          subTitle: 'Action',
          imageUri: '/icons/common/action.svg',
          status: 'healthy',
          onClick: () => {
            console.log(action);
          },
        },
      })),
    ];
  }, [actions, centerColumnX]);

  // Create edges connecting sources to actions, and actions to destinations
  const edges = useMemo(() => {
    const sourceToActionEdges = sources.map((_, sourceIndex) => {
      const actionIndex = sourceIndex % actions.length;
      return {
        id: `source-${sourceIndex}-to-action-${actionIndex}`,
        source: `source-${sourceIndex}`,
        target: `action-${actionIndex}`,
        style: { stroke: '#525252' },
      };
    });

    const actionToDestinationEdges = actions.flatMap((_, actionIndex) => {
      // Create an edge from each action to each destination
      return destinations.map((_, destinationIndex) => ({
        id: `action-${actionIndex}-to-destination-${destinationIndex}`,
        source: `action-${actionIndex}`,
        target: `destination-${destinationIndex}`,
        style: { stroke: '#525252' },
      }));
    });

    return [...sourceToActionEdges, ...actionToDestinationEdges];
  }, [sources, actions, destinations]);

  return (
    <ReactFlowProvider>
      <OverviewDataFlowWrapper ref={containerRef} className="nowheel">
        <NodeBaseDataFlow
          nodes={[...sourcesNode, ...destinationNode, ...actionsNode]}
          edges={edges}
        />
      </OverviewDataFlowWrapper>
    </ReactFlowProvider>
  );
}
