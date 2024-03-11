'use client';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { OverviewDataFlowWrapper } from './styled';
import { KeyvalDataFlow, KeyvalLoader } from '@/design.system';
import { useActions, useDestinations, useSources } from '@/hooks';
import { getEdges, groupSourcesNamespace, getNodes } from './utils';

const NAMESPACE_NODE_HEIGHT = 84;
const NAMESPACE_NODE_POSITION = 0;
const DESTINATION_NODE_HEIGHT = 136;
const DESTINATION_NODE_POSITION = 800;

export function DataFlowContainer() {
  const [containerHeight, setContainerHeight] = useState(0);

  const containerRef = useCallback((node: HTMLDivElement) => {
    if (node !== null) {
      setContainerHeight(node.getBoundingClientRect().height);
    }
  }, []);

  const { sources } = useSources();
  const { destinationList } = useDestinations();
  const { actions } = useActions();

  useEffect(() => {
    console.log({ sources });
    console.log({ destinationList });
    console.log({ actions });
  }, [actions, destinationList, sources]);
  const sourcesNodes = useMemo(() => {
    const groupedSources = groupSourcesNamespace(sources);

    const nodes = getNodes(
      containerHeight,
      groupedSources.length > 1 ? groupSourcesNamespace(sources) : sources,
      'namespace',
      NAMESPACE_NODE_HEIGHT,
      NAMESPACE_NODE_POSITION,
      true
    );

    return nodes;
  }, [sources, containerHeight]);

  const destinationsNodes = useMemo(
    () =>
      getNodes(
        containerHeight,
        destinationList,
        'destination',
        destinationList?.length > 1
          ? DESTINATION_NODE_HEIGHT
          : NAMESPACE_NODE_HEIGHT,
        DESTINATION_NODE_POSITION
      ),
    [destinationList, containerHeight]
  );

  const actionsNodes = useMemo(
    () =>
      getNodes(
        containerHeight,
        actions,
        'namespace',
        NAMESPACE_NODE_HEIGHT,
        DESTINATION_NODE_POSITION + 200
      ),
    [actions, containerHeight]
  );

  const edges = useMemo(() => {
    if (!destinationsNodes || !sourcesNodes) return [];

    return getEdges(destinationsNodes, sourcesNodes);
  }, [destinationsNodes, sourcesNodes]);

  if (!destinationsNodes || !sourcesNodes) {
    return <KeyvalLoader />;
  }

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <KeyvalDataFlow
        nodes={[...destinationsNodes, ...sourcesNodes]}
        edges={edges}
      />
    </OverviewDataFlowWrapper>
  );
}
