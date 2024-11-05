import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { STATUSES } from '@/types';
import { Handle, Position } from '@xyflow/react';
import { Status, Text } from '@/reuseable-components';
import { getStatusIcon } from '@/utils';

const BaseNodeContainer = styled.div<{ nodeWidth: number; isError?: boolean }>`
  width: ${({ nodeWidth }) => `${nodeWidth}px`};
  padding: 16px 24px 16px 16px;
  gap: 8px;
  display: flex;
  align-items: center;
  align-self: stretch;
  border-radius: 16px;
  cursor: pointer;
  background-color: ${({ isError, theme }) => (isError ? '#281515' : theme.colors.white_opacity['004'])};

  &:hover {
    background-color: ${({ isError, theme }) => (isError ? '#351515' : theme.colors.white_opacity['008'])};
  }
`;

const SourceIconWrapper = styled.div<{ isError?: boolean }>`
  display: flex;
  width: 36px;
  height: 36px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  background: ${({ isError }) =>
    `linear-gradient(180deg, ${isError ? 'rgba(237, 124, 124, 0.08)' : 'rgba(249, 249, 249, 0.06)'} 0%, ${
      isError ? 'rgba(237, 124, 124, 0.02)' : 'rgba(249, 249, 249, 0.02)'
    } 100%)`};
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

const Title = styled(Text)<{ nodeWidth: number }>`
  width: ${({ nodeWidth }) => `${nodeWidth - 75}px`};
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
  nodeWidth: number;
  isConnectable: boolean;
  data: NodeDataProps;
}

const BaseNode = ({ nodeWidth, isConnectable, data }: BaseNodeProps) => {
  const { type, status, title, subTitle, imageUri, monitors, isActive } = data;

  function renderHandles() {
    switch (type) {
      case 'source':
        return <Handle type='source' position={Position.Right} id='source-output' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />;
      case 'action':
        return (
          <>
            <Handle type='target' position={Position.Top} id='action-input' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
            <Handle type='source' position={Position.Bottom} id='action-output' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
          </>
        );
      case 'destination':
        return (
          <Handle type='target' position={Position.Left} id='destination-input' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />
        );
      default:
        return null;
    }
  }

  function renderMonitors() {
    if (!monitors) return null;

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
    if (typeof isActive !== 'boolean') return null;

    return (
      <FooterWrapper>
        <FooterText>{'·'}</FooterText>
        <Status isActive={isActive} withSmaller withSpecialFont />
      </FooterWrapper>
    );
  }

  const isError = status === STATUSES.UNHEALTHY;

  return (
    <BaseNodeContainer nodeWidth={nodeWidth} isError={isError}>
      <SourceIconWrapper>
        <Image src={imageUri || '/icons/common/folder.svg'} width={20} height={20} alt='source' />
      </SourceIconWrapper>
      <BodyWrapper>
        <Title nodeWidth={nodeWidth}>{title}</Title>
        <FooterWrapper>
          <FooterText>{subTitle}</FooterText>
          {renderMonitors()}
          {renderStatus()}
        </FooterWrapper>
      </BodyWrapper>
      {isError ? <Image src={getStatusIcon(false)} alt='' width={20} height={20} /> : null}
      {renderHandles()}
    </BaseNodeContainer>
  );
};

export default BaseNode;
