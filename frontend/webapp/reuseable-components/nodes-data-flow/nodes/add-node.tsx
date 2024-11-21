import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';
import { OVERVIEW_NODE_TYPES, STATUSES } from '@/types';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';

interface Props
  extends NodeProps<
    Node<
      {
        type: OVERVIEW_NODE_TYPES;
        status: STATUSES;
        title: string;
        subTitle: string;
      },
      'add'
    >
  > {
  nodeWidth: number;
}

const BaseNodeContainer = styled.div<{ $nodeWidth: Props['nodeWidth'] }>`
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
  padding: 16px 24px 16px 16px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 4px;
  align-self: stretch;
  cursor: pointer;
  background-color: transparent;
  border-radius: 16px;
  border: 1px dashed ${({ theme }) => theme.colors.border};

  &:hover {
    background-color: ${({ theme }) => theme.colors.white_opacity['004']};
  }
`;

const TitleWrapper = styled.div`
  display: flex;
  gap: 4px;
  align-items: center;
`;

const Title = styled(Text)`
  font-size: 14px;
  font-weight: 600;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-decoration-line: underline;
`;

const SubTitle = styled(Text)`
  font-size: 12px;
  color: ${({ theme }) => theme.text.grey};
  text-align: center;
`;

const AddNode: React.FC<Props> = ({ nodeWidth, data, id, isConnectable }) => {
  return (
    <BaseNodeContainer $nodeWidth={nodeWidth}>
      <TitleWrapper>
        <Image src='/icons/common/plus.svg' width={16} height={16} alt='plus' />
        <Title>{data.title}</Title>
      </TitleWrapper>
      <SubTitle>{data.subTitle}</SubTitle>
      <Handle type='target' position={Position.Left} id={`${id}-input`} isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
      <Handle type='source' position={Position.Right} id={`${id}-output`} isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
    </BaseNodeContainer>
  );
};

export default AddNode;
