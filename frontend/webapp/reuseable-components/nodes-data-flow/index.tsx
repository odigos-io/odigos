'use client';
import React, { useMemo } from 'react';
import '@xyflow/react/dist/style.css';
import AddNode from './nodes/add-node';
import BaseNode from './nodes/base-node';
import { ReactFlow } from '@xyflow/react';
import HeaderNode from './nodes/header-node';

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
    }),
    [columnWidth]
  );

  return (
    <div style={{ height: 'calc(100vh - 100px)' }}>
      <ReactFlow nodeTypes={nodeTypes} nodes={nodes} edges={edges} zoomOnScroll={false} onNodeClick={onNodeClick} />
    </div>
  );
}

export * from './builder';
