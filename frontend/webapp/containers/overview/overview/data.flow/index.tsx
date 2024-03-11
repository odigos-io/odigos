'use client';
import React, { useEffect, useState } from 'react';
import { OverviewDataFlowWrapper } from './styled';
import { KeyvalDataFlow, KeyvalLoader } from '@/design.system';
import { useActions, useDestinations, useSources } from '@/hooks';
import { buildFlowNodesAndEdges } from '@keyval-dev/design-system';

interface FlowNode {
  id: string;
  type: string;
  position: {
    x: number;
    y: number;
  };
  data: any;
}
interface FlowEdge {
  id: string;
  source: string;
  target: string;
  animated: boolean;
  label?: string;
  style?: Record<string, any>;
  data?: any;
}

export function DataFlowContainer() {
  const [nodes, setNodes] = useState<FlowNode[]>();
  const [edges, setEdges] = useState<FlowEdge[]>();

  const { sources } = useSources();
  const { destinationList } = useDestinations();
  const { actions } = useActions();

  useEffect(() => {
    if (!sources || !destinationList || !actions) return;

    const filteredActions = actions.filter(
      (action) => action.spec.disabled !== true
    );

    const { nodes, edges } = buildFlowNodesAndEdges(
      sources,
      destinationList,
      filteredActions
    );
    setNodes(nodes);
    setEdges(edges);
  }, [actions, destinationList, sources]);

  if (!nodes || !edges) {
    return <KeyvalLoader />;
  }

  return (
    <OverviewDataFlowWrapper>
      <KeyvalDataFlow nodes={nodes} edges={edges} />
    </OverviewDataFlowWrapper>
  );
}
