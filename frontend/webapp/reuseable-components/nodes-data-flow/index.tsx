'use client';
import React, { useMemo } from 'react';
import '@xyflow/react/dist/style.css';
import styled from 'styled-components';
import AddNode from './nodes/add-node';
import BaseNode from './nodes/base-node';
import GroupNode from './nodes/group-node';
import HeaderNode from './nodes/header-node';
import LabeledEdge from './edges/labeled-edge';
import { Controls, type Edge, type Node, ReactFlow } from '@xyflow/react';

interface Props {
  nodes: Node[];
  edges: Edge[];
  onNodeClick?: (event: React.MouseEvent, object: Node) => void;
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
    border: 1px solid ${({ theme }) => theme.colors.border} !important;
    background-color: ${({ theme }) => theme.colors.dropdown_bg};
    path {
      fill: #fff;
    }
    &:hover {
      background-color: ${({ theme }) => theme.colors.dropdown_bg_2};
    }
  }
`;

export const NodeBaseDataFlow: React.FC<Props> = ({ nodes, edges, onNodeClick, nodeWidth }) => {
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
          <Controls
            position='bottom-left'
            orientation='horizontal'
            showInteractive={false}
            showZoom
            showFitView
            fitViewOptions={{
              duration: 300,
              padding: 0.02,
              includeHiddenNodes: true,
            }}
          />
        </ControllerWrapper>
      </ReactFlow>
    </FlowWrapper>
  );
};

export * from './builder';
