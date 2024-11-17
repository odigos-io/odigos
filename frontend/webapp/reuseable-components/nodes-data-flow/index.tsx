'use client';
import React, { useMemo } from 'react';
import '@xyflow/react/dist/style.css';
import AddNode from './nodes/add-node';
import BaseNode from './nodes/base-node';
import { Controls, ReactFlow } from '@xyflow/react';
import GroupNode from './nodes/group-node';
import HeaderNode from './nodes/header-node';
import LabeledEdge from './edges/labeled-edge';
import styled from 'styled-components';

interface NodeBaseDataFlowProps {
  nodes: any[];
  edges: any[];
  onNodeClick?: (event: React.MouseEvent, object: any) => void;
  nodeWidth: number;
}

const FlowWrapper = styled.div`
  height: calc(100vh - 160px);
  .react-flow__attribution {
    visibility: hidden;
  }
`;

const ControllerWrapper = styled.div`
  button {
    padding: 8px;
    margin: 8px;
    border-radius: 8px;
    border: 1px solid ${({ theme }) => theme.colors.border};
    background-color: ${({ theme }) => theme.colors.dropdown_bg};
    path {
      fill: #fff;
    }
    &:hover {
      background-color: ${({ theme }) => theme.colors.dropdown_bg_2};
    }
  }
`;

export function NodeBaseDataFlow({ nodes, edges, onNodeClick, nodeWidth }: NodeBaseDataFlowProps) {
  const nodeTypes = useMemo(
    () => ({
      header: (props) => <HeaderNode {...props} nodeWidth={nodeWidth} />,
      add: (props) => <AddNode {...props} nodeWidth={nodeWidth} />,
      base: (props) => <BaseNode {...props} nodeWidth={nodeWidth} />,
      group: GroupNode,
    }),
    [nodeWidth],
  );

  const edgeTypes = useMemo(
    () => ({
      labeled: LabeledEdge,
    }),
    [],
  );

  return (
    <FlowWrapper>
      <ReactFlow nodes={nodes} nodeTypes={nodeTypes} edges={edges} edgeTypes={edgeTypes} onNodeClick={onNodeClick} zoomOnScroll={false} fitView={false}>
        <ControllerWrapper>
          <Controls position='bottom-left' orientation='horizontal' showZoom showFitView showInteractive={false} />
        </ControllerWrapper>
      </ReactFlow>
    </FlowWrapper>
  );
}

export * from './builder';
