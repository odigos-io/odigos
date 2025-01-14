'use client';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import styled, { useTheme } from 'styled-components';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { NodeDataFlow } from '@/reuseable-components';
import { MultiSourceControl } from '../multi-source-control';
import { OverviewActionsMenu } from '../overview-actions-menu';
import { type Edge, useEdgesState, useNodesState, type Node, applyNodeChanges } from '@xyflow/react';
import { useComputePlatform, useContainerSize, useMetrics, useNodeDataFlowHandlers, useSourceCRUD } from '@/hooks';

import { buildEdges } from './build-edges';
import { getEntityCounts } from './get-entity-counts';
import { getNodePositions } from './get-node-positions';
import { buildRuleNodes } from './build-rule-nodes';
import { buildActionNodes } from './build-action-nodes';
import { buildDestinationNodes } from './build-destination-nodes';
import { buildSourceNodes } from './build-source-nodes';
import nodeConfig from './node-config.json';

export * from './get-entity-counts';
export * from './get-node-positions';
export { nodeConfig };

const Container = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

export default function OverviewDataFlowContainer() {
  const theme = useTheme();

  const [scrollYOffset, setScrollYOffset] = useState(0);

  const { handleNodeClick } = useNodeDataFlowHandlers();
  const { containerRef, containerWidth, containerHeight } = useContainerSize();
  const positions = useMemo(() => getNodePositions({ containerWidth }), [containerWidth]);

  const { metrics } = useMetrics();
  const { sources, filteredSources } = useSourceCRUD();
  const { data, filteredData, loading } = useComputePlatform(); // TODO: remove this in favor of CRUD hooks

  const unfilteredCounts = useMemo(
    () =>
      getEntityCounts({
        sources,
        destinations: data?.computePlatform.destinations,
        actions: data?.computePlatform.actions,
        instrumentationRules: data?.computePlatform.instrumentationRules,
      }),
    [sources, data],
  );

  const ruleNodes = useMemo(
    () =>
      buildRuleNodes({
        loading,
        entities: filteredData?.computePlatform.instrumentationRules || [],
        positions,
        unfilteredCounts,
      }),
    [loading, filteredData?.computePlatform.instrumentationRules, positions, unfilteredCounts.rule],
  );
  const actionNodes = useMemo(
    () =>
      buildActionNodes({
        loading,
        entities: filteredData?.computePlatform.actions || [],
        positions,
        unfilteredCounts,
      }),
    [loading, filteredData?.computePlatform.actions, positions, unfilteredCounts.action],
  );
  const destinationNodes = useMemo(
    () =>
      buildDestinationNodes({
        loading,
        entities: filteredData?.computePlatform.destinations || [],
        positions,
        unfilteredCounts,
      }),
    [loading, filteredData?.computePlatform.destinations, positions, unfilteredCounts.destination],
  );
  const sourceNodes = useMemo(
    () =>
      buildSourceNodes({
        loading,
        entities: filteredSources,
        positions,
        unfilteredCounts,
        containerHeight,
        onScroll: ({ scrollTop }) => setScrollYOffset(scrollTop),
      }),
    [loading, filteredSources, positions, unfilteredCounts.source, containerHeight],
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(([] as Node[]).concat(actionNodes, ruleNodes, sourceNodes, destinationNodes));
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as Edge[]);

  const handleNodeState = useCallback((prevNodes: Node[], currNodes: Node[], key: OVERVIEW_ENTITY_TYPES, yOffset?: number) => {
    const filtered = [...prevNodes].filter(({ id }) => id.split('-')[0] !== key);

    if (!!yOffset) {
      const changed = applyNodeChanges(
        currNodes.filter((node) => node.extent === 'parent').map((node) => ({ id: node.id, type: 'position', position: { ...node.position, y: node.position.y - yOffset } })),
        prevNodes,
      );

      return changed;
    } else {
      filtered.push(...currNodes);
    }

    return filtered;
  }, []);

  useEffect(() => setNodes((prev) => handleNodeState(prev, ruleNodes, OVERVIEW_ENTITY_TYPES.RULE)), [ruleNodes]);
  useEffect(() => setNodes((prev) => handleNodeState(prev, actionNodes, OVERVIEW_ENTITY_TYPES.ACTION)), [actionNodes]);
  useEffect(() => setNodes((prev) => handleNodeState(prev, destinationNodes, OVERVIEW_ENTITY_TYPES.DESTINATION)), [destinationNodes]);
  useEffect(() => setNodes((prev) => handleNodeState(prev, sourceNodes, OVERVIEW_ENTITY_TYPES.SOURCE, scrollYOffset)), [sourceNodes, scrollYOffset]);
  useEffect(() => setEdges(buildEdges({ theme, nodes, metrics, containerHeight })), [theme, nodes, metrics, containerHeight]);

  return (
    <Container ref={containerRef}>
      <OverviewActionsMenu />
      <MultiSourceControl />
      <NodeDataFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} />
    </Container>
  );
}
