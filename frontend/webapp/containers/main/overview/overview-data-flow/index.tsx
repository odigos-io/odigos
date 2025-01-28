'use client';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import styled, { useTheme } from 'styled-components';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { NodeDataFlow } from '@/reuseable-components';
import { MultiSourceControl } from '../multi-source-control';
import { OverviewActionsMenu } from '../overview-actions-menu';
import { type Edge, useEdgesState, useNodesState, type Node, applyNodeChanges } from '@xyflow/react';
import { useActionCRUD, useContainerSize, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNodeDataFlowHandlers, useSourceCRUD } from '@/hooks';

import { buildEdges } from './build-edges';
import { getNodePositions } from './get-node-positions';
import { buildRuleNodes } from './build-rule-nodes';
import { buildActionNodes } from './build-action-nodes';
import { buildDestinationNodes } from './build-destination-nodes';
import { buildSourceNodes } from './build-source-nodes';
import nodeConfig from './node-config.json';

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
  const { actions, filteredActions, loading: actLoad } = useActionCRUD();
  const { sources, filteredSources, loading: srcLoad } = useSourceCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad } = useInstrumentationRuleCRUD();

  const ruleNodes = useMemo(
    () =>
      buildRuleNodes({
        loading: ruleLoad,
        entities: filteredInstrumentationRules,
        unfilteredCount: instrumentationRules.length,
        positions,
      }),
    [ruleLoad, instrumentationRules, filteredInstrumentationRules, positions],
  );
  const actionNodes = useMemo(
    () =>
      buildActionNodes({
        loading: actLoad,
        entities: filteredActions,
        unfilteredCount: actions.length,
        positions,
      }),
    [actLoad, actions, filteredActions, positions],
  );
  const destinationNodes = useMemo(
    () =>
      buildDestinationNodes({
        loading: destLoad,
        entities: filteredDestinations,
        unfilteredCount: destinations.length,
        positions,
      }),
    [destLoad, destinations, filteredDestinations, positions],
  );
  const sourceNodes = useMemo(
    () =>
      buildSourceNodes({
        loading: srcLoad,
        entities: filteredSources,
        unfilteredCount: sources.length,
        positions,
        containerHeight,
        onScroll: ({ scrollTop }) => setScrollYOffset(scrollTop),
      }),
    [srcLoad, sources, filteredSources, positions, containerHeight],
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
      <NodeDataFlow
        nodes={nodes}
        edges={edges}
        onNodeClick={handleNodeClick}
        onNodesChange={(changes) => setTimeout(() => onNodesChange(changes))} // Timeout is needed to fix this error: "ResizeObserver loop completed with undelivered notifications."
        onEdgesChange={(changes) => setTimeout(() => onEdgesChange(changes))} // Timeout is needed to fix this error: "ResizeObserver loop completed with undelivered notifications."
      />
    </Container>
  );
}
