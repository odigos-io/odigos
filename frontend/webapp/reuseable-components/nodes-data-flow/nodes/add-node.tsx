import React, { Fragment } from 'react';
import { PlusIcon } from '@/assets';
import styled from 'styled-components';
import { usePendingStore } from '@/store';
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

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth'] }>`
  // negative width applied here because of the padding left&right
  width: ${({ $nodeWidth }) => `${$nodeWidth - 40}px`};
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

const AddNode: React.FC<Props> = ({ id: nodeId, data }) => {
  const { nodeWidth, title, subTitle } = data;

  const { isThisPending } = usePendingStore();
  const entity = nodeId.split('-')[0] as OVERVIEW_ENTITY_TYPES;
  const isPending = isThisPending({ entityType: entity });

  return (
    <Container $nodeWidth={nodeWidth} className='nowheel nodrag'>
      <TitleWrapper>
        {isPending ? (
          <FadeLoader />
        ) : (
          <Fragment>
            <PlusIcon />
            <Title>{title}</Title>
          </Fragment>
        )}
      </TitleWrapper>
      <SubTitle>{isPending ? `Adding ${entity}...` : subTitle}</SubTitle>

      <Handle type='target' position={Position.Left} style={{ visibility: 'hidden' }} />
      <Handle type='source' position={Position.Right} style={{ visibility: 'hidden' }} />
    </Container>
  );
};

export default AddNode;
