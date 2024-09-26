import Image from 'next/image';
import React, { memo } from 'react';
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

export interface NodeDataProps {
  type: 'source' | 'action' | 'destination';
  title: string;
  subTitle: string;
  imageUri: string;
  monitors?: string[];
  status: 'healthy' | 'unhealthy';
  onClick: () => void;
}

interface BaseNodeProps {
  data: NodeDataProps;
  isConnectable: boolean;
}

export default memo(({ isConnectable, data }: BaseNodeProps) => {
  const { onClick } = data;

  return (
    <BaseNodeContainer onClick={onClick}>
      <TitleWrapper>
        <Image
          src={'/icons/common/plus.svg'}
          width={16}
          height={16}
          alt="plus"
        />
        <Title>{'ADD ACTION'}</Title>
      </TitleWrapper>
      <SubTitle>{'Add first action to modify the OpenTelemetry data'}</SubTitle>
      <Handle
        type="target"
        position={Position.Left}
        id="action-input"
        isConnectable={isConnectable}
        style={{ visibility: 'hidden' }}
      />
      <Handle
        type="source"
        position={Position.Right}
        id="action-output"
        isConnectable={isConnectable}
        style={{ visibility: 'hidden' }}
      />
    </BaseNodeContainer>
  );
});
