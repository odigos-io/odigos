'use client';
import React from 'react';
import '@xyflow/react/dist/style.css';
import BaseNode from './nodes/base-node';
import headerNode from './nodes/header-node';
import { ReactFlow } from '@xyflow/react';

const nodeTypes = {
  base: BaseNode,
  header: headerNode,
};

interface NodeBaseDataFlowProps {
  nodes: any[];
  edges: any[];
}

export function NodeBaseDataFlow({ nodes, edges }: NodeBaseDataFlowProps) {
  return (
    <div style={{ height: 'calc(100vh - 100px)' }}>
      <ReactFlow
        nodeTypes={nodeTypes}
        nodes={nodes}
        edges={edges}
        zoomOnScroll={false}
      />
    </div>
  );
}
