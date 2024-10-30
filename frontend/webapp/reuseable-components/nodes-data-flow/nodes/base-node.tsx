import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import type { STATUSES } from '@/types';
import { Handle, Position } from '@xyflow/react';
import { Status, Text } from '@/reuseable-components';

const BaseNodeContainer = styled.div<{ columnWidth: number }>`
  width: ${({ columnWidth }) => `${columnWidth}px`};
  padding: 16px 24px 16px 16px;
  gap: 8px;
  display: flex;
  align-items: center;
  align-self: stretch;
  border-radius: 16px;
  cursor: pointer;
  background-color: ${({ theme }) => theme.colors.white_opacity['004']};

  &:hover {
    background-color: ${({ theme }) => theme.colors.white_opacity['008']};
  }
`;

const SourceIconWrapper = styled.div`
  display: flex;
  width: 36px;
  height: 36px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  background: linear-gradient(180deg, rgba(249, 249, 249, 0.06) 0%, rgba(249, 249, 249, 0.02) 100%);
`;

const BodyWrapper = styled.div`
  display: flex;
  flex-direction: column;
  height: 36px;
  justify-content: space-between;
`;

const FooterWrapper = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
`;

const Title = styled(Text)<{ columnWidth: number }>`
  width: ${({ columnWidth }) => `${columnWidth - 42}px`};
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
`;

const FooterText = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
`;

export interface NodeDataProps {
  id: string;
  type: 'source' | 'action' | 'destination';
  status: STATUSES;
  title: string;
  subTitle: string;
  imageUri: string;
  monitors?: string[];
  isActive?: boolean;
}

interface BaseNodeProps {
  id: string;
  isConnectable: boolean;
  data: NodeDataProps;
  columnWidth: number;
}

const BaseNode = ({ isConnectable, data, columnWidth }: BaseNodeProps) => {
  const { title, subTitle, imageUri, type, monitors, isActive } = data;

  function renderHandles() {
    switch (type) {
      case 'source':
        return (
          <>
            {/* Source nodes have an output handle */}
            <Handle type='source' position={Position.Right} id='source-output' style={{ visibility: 'hidden' }} isConnectable={isConnectable} />
          </>
        );
      case 'action':
        return (
          <>
            {/* Action nodes have both input and output handles */}
            <Handle type='target' position={Position.Left} id='action-input' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
            <Handle type='source' position={Position.Right} id='action-output' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
          </>
        );
      case 'destination':
        return (
          <>
            {/* Destination nodes only have an input handle */}
            <Handle type='target' position={Position.Left} id='destination-input' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
          </>
        );
      default:
        return null;
    }
  }

  function renderMonitors() {
    if (!monitors) {
      return null;
    }

    return (
      <FooterWrapper>
        <FooterText>{'·'}</FooterText>
        {monitors.map((monitor, index) => (
          <Image key={index} src={`/icons/monitors/${monitor}.svg`} width={10} height={10} alt={monitor} />
        ))}
      </FooterWrapper>
    );
  }

  function renderStatus() {
    if (typeof isActive !== 'boolean') {
      return null;
    }

    return (
      <FooterWrapper>
        <FooterText>{'·'}</FooterText>
        <Status isActive={isActive} withSmaller withSpecialFont />
      </FooterWrapper>
    );
  }

  return (
    <BaseNodeContainer columnWidth={columnWidth}>
      <SourceIconWrapper>
        <Image src={imageUri || '/icons/common/folder.svg'} width={20} height={20} alt='source' />
      </SourceIconWrapper>
      <BodyWrapper>
        <Title columnWidth={columnWidth}>{title}</Title>
        <FooterWrapper>
          <FooterText>{subTitle}</FooterText>
          {renderMonitors()}
          {renderStatus()}
        </FooterWrapper>
      </BodyWrapper>
      {renderHandles()}
    </BaseNodeContainer>
  );
};

export default BaseNode;
