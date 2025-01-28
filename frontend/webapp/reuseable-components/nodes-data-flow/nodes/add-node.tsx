import React, { Fragment } from 'react';
import { PlusIcon } from '@/assets';
import styled from 'styled-components';
import { usePendingStore } from '@/store';
import { FlexColumn, FlexRow, hexPercentValues } from '@/styles';
import { FadeLoader, Text } from '@/reuseable-components';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';
import { NODE_TYPES, OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, STATUSES } from '@/types';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;

        type: OVERVIEW_NODE_TYPES;
        status: STATUSES;
        title: string;
        subTitle: string;
      },
      NODE_TYPES.ADD
    >
  > {}

const Container = styled(FlexColumn)<{ $nodeWidth: Props['data']['nodeWidth'] }>`
  min-height: 50px;
  // negative width applied here because of the padding left&right
  width: ${({ $nodeWidth }) => `${$nodeWidth - 40}px`};
  padding: 16px 24px 16px 16px;
  align-items: center;
  justify-content: center;
  gap: 4px;
  align-self: stretch;
  cursor: pointer;
  background-color: transparent;
  border-radius: 16px;
  border: 1px dashed ${({ theme }) => theme.colors.border};

  &:hover {
    background-color: ${({ theme }) => theme.colors.dropdown_bg_2 + hexPercentValues['030']};
  }
`;

const TitleWrapper = styled(FlexRow)`
  gap: 4px;
`;

const Title = styled(Text)`
  font-size: 14px;
  font-weight: 600;
`;

const SubTitle = styled(Text)`
  font-size: 12px;
  color: ${({ theme }) => theme.text.grey};
  text-align: center;
`;

const AddNode: React.FC<Props> = ({ id: nodeId, data }) => {
  const { nodeWidth, title, subTitle } = data;

  const { isThisPending } = usePendingStore();
  const entity = nodeId.split('-')[0] as OVERVIEW_ENTITY_TYPES;
  const isPending = isThisPending({ entityType: entity });

  return (
    <Container $nodeWidth={nodeWidth} className='nowheel nodrag'>
      {isPending ? (
        <Fragment>
          <TitleWrapper>
            <FadeLoader />
            <Title family='secondary'>adding {entity}s</Title>
          </TitleWrapper>
          <SubTitle>Just a few more seconds...</SubTitle>
        </Fragment>
      ) : (
        <Fragment>
          <TitleWrapper>
            <PlusIcon />
            <Title family='secondary' decoration='underline'>
              {title}
            </Title>
          </TitleWrapper>
          <SubTitle>{subTitle}</SubTitle>
        </Fragment>
      )}

      <Handle type='target' position={Position.Left} style={{ visibility: 'hidden' }} />
      <Handle type='source' position={Position.Right} style={{ visibility: 'hidden' }} />
    </Container>
  );
};

export default AddNode;
