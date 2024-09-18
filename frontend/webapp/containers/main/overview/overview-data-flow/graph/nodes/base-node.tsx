import { Text } from '@/reuseable-components';
import { Handle, Position } from '@xyflow/react';
import Image from 'next/image';
import React, { memo } from 'react';
import styled from 'styled-components';

const BaseNodeContainer = styled.div`
  display: flex;
  padding: 16px 24px 16px 16px;
  align-items: center;
  gap: 8px;
  align-self: stretch;
  border-radius: 16px;
  width: 296px;
  cursor: pointer;
  background-color: ${({ theme }) => theme.colors.white_opacity['004']};

  &:hover {
    background-color: ${({ theme }) => theme.colors.white_opacity['008']};
  }
`;

const ListItemContent = styled.div`
  display: flex;
  gap: 12px;
`;

const SourceIconWrapper = styled.div`
  display: flex;
  width: 36px;
  height: 36px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  background: linear-gradient(
    180deg,
    rgba(249, 249, 249, 0.06) 0%,
    rgba(249, 249, 249, 0.02) 100%
  );
`;

const TextWrapper = styled.div`
  display: flex;
  flex-direction: column;
  height: 36px;
  justify-content: space-between;
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
  const { title, subTitle, imageUri, status, onClick, type } = data;

  function renderHandles() {
    switch (type) {
      case 'source':
        return (
          <>
            {/* Source nodes have an output handle */}
            <Handle
              type="source"
              position={Position.Right}
              id="source-output"
              style={{ visibility: 'hidden' }}
              isConnectable={isConnectable}
            />
          </>
        );
      case 'action':
        return (
          <>
            {/* Action nodes have both input and output handles */}
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
          </>
        );
      case 'destination':
        return (
          <>
            {/* Destination nodes only have an input handle */}
            <Handle
              style={{ visibility: 'hidden' }}
              type="target"
              position={Position.Left}
              id="destination-input"
              isConnectable={isConnectable}
            />
          </>
        );
      default:
        return null;
    }
  }

  return (
    <BaseNodeContainer onClick={onClick}>
      <SourceIconWrapper>
        <Image
          src={imageUri || '/icons/common/folder.svg'}
          width={20}
          height={20}
          alt="source"
        />
      </SourceIconWrapper>
      <TextWrapper>
        <Text>{title}</Text>
        <Text opacity={0.8} size={10}>
          {subTitle}
        </Text>
      </TextWrapper>
      {/* Conditionally render handles based on node type */}
      {renderHandles()}
    </BaseNodeContainer>
  );
});
