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
  columnWidth: number;
}

export function NodeBaseDataFlow({ nodes, edges, onNodeClick, columnWidth }: NodeBaseDataFlowProps) {
  const nodeTypes = useMemo(
    () => ({
      header: (props) => <HeaderNode {...props} columnWidth={columnWidth} />,
      add: (props) => <AddNode {...props} columnWidth={columnWidth} />,
      base: (props) => <BaseNode {...props} columnWidth={columnWidth} />,
      group: GroupNode,
    }),
    [columnWidth]
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
