'use client';
import React, { useEffect } from 'react';
import '@xyflow/react/dist/style.css';
import BaseNode from './nodes/base-node';
import headerNode from './nodes/header-node';
import { ReactFlow, useReactFlow } from '@xyflow/react';

const nodeTypes = {
  base: BaseNode,
  header: headerNode,
};

interface NodeBaseDataFlowProps {
  nodes: any[];
  edges: any[];
}

export function NodeBaseDataFlow({ nodes, edges }: NodeBaseDataFlowProps) {
  const { fitView } = useReactFlow();

  // useEffect(() => {
  //   setTimeout(() => {
  //     fitView();
  //   }, 100);
  // }, [fitView, nodes, edges]);
  return (
    <div style={{ height: 'calc(100vh - 100px)' }}>
      <ReactFlow
        nodeTypes={nodeTypes}
        nodes={nodes}
        edges={edges}
        // panOnDrag={false}
        zoomOnScroll={false}
      />
    </div>
  );
}
