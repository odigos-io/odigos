'use client';
import React, { useMemo } from 'react';
import '@xyflow/react/dist/style.css';
import AddNode from './nodes/add-node';
import BaseNode from './nodes/base-node';
import { ReactFlow } from '@xyflow/react';
import GroupNode from './nodes/group-node';
import HeaderNode from './nodes/header-node';
import LabeledEdge from './edges/labeled-edge';

interface NodeBaseDataFlowProps {
  nodes: any[];
  edges: any[];
  onNodeClick?: (event: React.MouseEvent, object: any) => void;
  nodeWidth: number;
}

export function NodeBaseDataFlow({ nodes, edges, onNodeClick, nodeWidth }: NodeBaseDataFlowProps) {
  const nodeTypes = useMemo(
    () => ({
      header: (props) => <HeaderNode {...props} nodeWidth={nodeWidth} />,
      add: (props) => <AddNode {...props} nodeWidth={nodeWidth} />,
      base: (props) => <BaseNode {...props} nodeWidth={nodeWidth} />,
      group: GroupNode,
    }),
    [nodeWidth]
  );

  const edgeTypes = useMemo(
    () => ({
      labeled: LabeledEdge,
    }),
    []
  );

  return (
    <div style={{ height: 'calc(100vh - 100px)' }}>
      <ReactFlow
        nodes={nodes}
        nodeTypes={nodeTypes}
        edges={edges}
        edgeTypes={edgeTypes}
        onNodeClick={onNodeClick}
        zoomOnScroll={false}
        fitView={false}
      />
    </div>
  );
}

export * from './builder';
