import React from 'react';
import styled from 'styled-components';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;
        nodeHeight: number;
      },
      'scroll'
    >
  > {}

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth']; $nodeHeight: Props['data']['nodeHeight'] }>`
  width: ${({ $nodeWidth }) => $nodeWidth}px;
  height: ${({ $nodeHeight }) => $nodeHeight}px;
  background: transparent;
  border: 1px dashed red;
  overflow-y: scroll;
`;

const ScrollNode: React.FC<Props> = ({ data }) => {
  const { nodeWidth, nodeHeight } = data;

  return (
    <Container $nodeWidth={nodeWidth} $nodeHeight={nodeHeight} className='nowheel nodrag'>
      <Handle type='source' position={Position.Right} style={{ visibility: 'hidden' }} />
      <Handle type='target' position={Position.Left} style={{ visibility: 'hidden' }} />
    </Container>
  );
};

export default ScrollNode;
