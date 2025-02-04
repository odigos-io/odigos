import React from 'react';
import { NODE_TYPES } from '@/types';
import styled from 'styled-components';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;
        nodeHeight: number;
      },
      NODE_TYPES.EDGED
    >
  > {}

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth']; $nodeHeight: Props['data']['nodeHeight'] }>`
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
  height: ${({ $nodeHeight }) => `${$nodeHeight}px`};
  opacity: 0;
`;

const EdgedNode: React.FC<Props> = ({ data }) => {
  const { nodeWidth, nodeHeight } = data;

  return (
    <Container $nodeWidth={nodeWidth} $nodeHeight={nodeHeight}>
      <Handle type='source' position={Position.Right} style={{ visibility: 'hidden' }} />
      <Handle type='target' position={Position.Left} style={{ visibility: 'hidden' }} />
    </Container>
  );
};

export default EdgedNode;
