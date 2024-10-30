'use client';
import React from 'react';
import '@xyflow/react/dist/style.css';
import BaseNode from './nodes/base-node';
import { ReactFlow } from '@xyflow/react';
import headerNode from './nodes/header-node';
import AddNode from './nodes/add-node';

const nodeTypes = {
  header: headerNode,
  add: AddNode,
  base: BaseNode,
};

interface NodeBaseDataFlowProps {
  nodes: any[];
  edges: any[];
  onNodeClick?: (event: React.MouseEvent, object: any) => void;
}

export function NodeBaseDataFlow({ nodes, edges, onNodeClick }: NodeBaseDataFlowProps) {
  return (
    <div style={{ height: 'calc(100vh - 100px)' }}>
      <ReactFlow nodeTypes={nodeTypes} nodes={nodes} edges={edges} zoomOnScroll={false} onNodeClick={onNodeClick} />
    </div>
  );
}

export * from './builder';
