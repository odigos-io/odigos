import React from 'react';
import Image from 'next/image';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';
import { Handle, Position } from '@xyflow/react';
import { Checkbox, Status, Text } from '@/reuseable-components';
import { type ActionDataParsed, type ActualDestination, type InstrumentationRuleSpec, type K8sActualSource, STATUSES } from '@/types';

export interface NodeDataProps {
  id: string;
  type: 'source' | 'action' | 'destination';
  status: STATUSES;
  title: string;
  subTitle: string;
  imageUri: string;
  monitors?: string[];
  isActive?: boolean;
  raw: InstrumentationRuleSpec | K8sActualSource | ActionDataParsed | ActualDestination;
}

interface BaseNodeProps {
  id: string;
  nodeWidth: number;
  isConnectable: boolean;
  data: NodeDataProps;
}

const Container = styled.div<{ $nodeWidth: number; $isError?: boolean }>`
  display: flex;
  align-items: center;
  align-self: stretch;
  gap: 8px;
  padding: 16px 24px 16px 16px;
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
  border-radius: 16px;
  cursor: pointer;
  background-color: ${({ $isError, theme }) => ($isError ? '#281515' : theme.colors.white_opacity['004'])};
  &:hover {
    background-color: ${({ $isError, theme }) => ($isError ? '#351515' : theme.colors.white_opacity['008'])};
  }
`;

const IconWrapper = styled.div<{ $isError?: boolean }>`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: ${({ $isError }) =>
    `linear-gradient(180deg, ${$isError ? 'rgba(237, 124, 124, 0.08)' : 'rgba(249, 249, 249, 0.06)'} 0%, ${$isError ? 'rgba(237, 124, 124, 0.02)' : 'rgba(249, 249, 249, 0.02)'} 100%)`};
`;

const BodyWrapper = styled.div`
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 36px;
`;

const Title = styled(Text)<{ $nodeWidth: number }>`
  max-width: ${({ $nodeWidth }) => `${Math.floor($nodeWidth * 0.5)}px`};
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
`;

const FooterWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const FooterText = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
`;

const ActionsWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
`;

const BaseNode = ({ nodeWidth, isConnectable, data }: BaseNodeProps) => {
  const { type, status, title, subTitle, imageUri, monitors, isActive, raw } = data;
  const isError = status === STATUSES.UNHEALTHY;

  const { configuredSources, setConfiguredSources } = useAppStore((state) => state);

  const renderHandles = () => {
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
        return <Handle type='target' position={Position.Left} id='destination-input' isConnectable={isConnectable} style={{ visibility: 'hidden' }} />;
      default:
        return null;
    }
  };

  const renderMonitors = () => {
    if (!monitors) return null;

    return (
      <FooterWrapper>
        <FooterText>{'·'}</FooterText>
        {monitors.map((monitor, index) => (
          <Image key={index} src={`/icons/monitors/${monitor}.svg`} width={10} height={10} alt={monitor} />
        ))}
      </FooterWrapper>
    );
  };

  const renderStatus = () => {
    if (typeof isActive !== 'boolean') return null;

    return (
      <FooterWrapper>
        <FooterText>{'·'}</FooterText>
        <Status isActive={isActive} withSmaller withSpecialFont />
      </FooterWrapper>
    );
  };

  const renderActions = () => {
    const getSourceLocation = () => {
      const { namespace, name, kind } = raw as K8sActualSource;
      const selected = { ...configuredSources };

      if (!selected[namespace]) selected[namespace] = [];

      const index = selected[namespace].findIndex((x) => x.name === name && x.kind === kind);

      return { index, namespace, selected };
    };

    const onSelectSource = () => {
      const { index, namespace, selected } = getSourceLocation();

      if (index === -1) {
        selected[namespace].push(raw as K8sActualSource);
      } else {
        selected[namespace].splice(index, 1);
      }

      setConfiguredSources(selected);
    };

    return (
      <>
        {/* TODO: handle instrumentation rules for sources */}
        {isError ? (
          <Image src={getStatusIcon('error')} alt='' width={20} height={20} />
        ) : // : type === 'source' && SOME_INDICATOR_THAT_THIS_IS_INSTRUMENTED ? ( <Image src={getEntityIcon(OVERVIEW_ENTITY_TYPES.RULE)} alt='' width={18} height={18} /> )
        null}

        {type === 'source' ? <Checkbox initialValue={getSourceLocation().index !== -1} onChange={onSelectSource} /> : null}
      </>
    );
  };

  return (
    <Container $nodeWidth={nodeWidth} $isError={isError}>
      <IconWrapper $isError={isError}>
        <Image src={imageUri || '/icons/common/folder.svg'} width={20} height={20} alt='source' />
      </IconWrapper>

      <BodyWrapper>
        <Title $nodeWidth={nodeWidth}>{title}</Title>
        <FooterWrapper>
          <FooterText>{subTitle}</FooterText>
          {renderMonitors()}
          {renderStatus()}
        </FooterWrapper>
      </BodyWrapper>

      <ActionsWrapper>{renderActions()}</ActionsWrapper>

      {renderHandles()}
    </Container>
  );
};

export default BaseNode;
