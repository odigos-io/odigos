import React from 'react';
import { NODE_TYPES } from '@/types';
import styled from 'styled-components';
import { type Node, type NodeProps } from '@xyflow/react';
import { SkeletonLoader } from '@/reuseable-components/skeleton-loader';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;
        size: number;
      },
      NODE_TYPES.SKELETON
    >
  > {}

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth'] }>`
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
`;

const SkeletonNode: React.FC<Props> = ({ id: nodeId, data }) => {
  const { nodeWidth, size } = data;

  return (
    <Container data-id={nodeId} $nodeWidth={nodeWidth} className='nowheel nodrag'>
      <SkeletonLoader size={size} />
    </Container>
  );
};

export default SkeletonNode;
