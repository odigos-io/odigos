import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';
import { Handle, Position } from '@xyflow/react';

const BaseNodeContainer = styled.div`
  display: flex;
  width: 296px;
  padding: 16px 24px 16px 16px;
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
`;

interface BaseNodeProps {
  data: Record<string, any>;

  id: string;
  parentId?: any;
  type: string;

  isConnectable: boolean;
  selectable: boolean;
  selected?: any;
  deletable: boolean;
  draggable: boolean;
  dragging: boolean;
  dragHandle?: any;

  width: number;
  height: number;
  zIndex: number;
  positionAbsoluteX: number;
  positionAbsoluteY: number;
  sourcePosition?: any;
  targetPosition?: any;
}

const AddNode = ({ isConnectable, data }: BaseNodeProps) => {
  return (
    <BaseNodeContainer>
      <TitleWrapper>
        <Image src='/icons/common/plus.svg' width={16} height={16} alt='plus' />
        <Title>{data.title}</Title>
      </TitleWrapper>
      <SubTitle>{data.subTitle}</SubTitle>
      <Handle type='target' position={Position.Left} id='add-node-input' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
      <Handle type='source' position={Position.Right} id='add-node-output' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
    </BaseNodeContainer>
  );
};

export default AddNode;
