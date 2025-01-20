import React, { useMemo } from 'react';
import styled from 'styled-components';
import { usePaginatedSources, useSourceCRUD } from '@/hooks';
import type { Node, NodeProps } from '@xyflow/react';
import { useAppStore, usePendingStore } from '@/store';
import { K8sActualSource, NODE_TYPES, OVERVIEW_ENTITY_TYPES } from '@/types';
import { Badge, Checkbox, FadeLoader, Text } from '@/reuseable-components';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;
        icon: string;
        title: string;
        tagValue: number;
      },
      NODE_TYPES.HEADER
    >
  > {}

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth'] }>`
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
  padding: 12px 0px 16px 0px;
  gap: 8px;
  display: flex;
  align-items: center;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
`;

const Title = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
`;

const ActionsWrapper = styled.div`
  margin-left: auto;
  margin-right: 16px;
`;

const HeaderNode: React.FC<Props> = ({ data }) => {
  const { nodeWidth, title, icon: Icon, tagValue } = data;
  const isSources = title === 'Sources';

  const { configuredSources, setConfiguredSources } = useAppStore();
  const { sourcesFetching } = usePaginatedSources();
  const { isThisPending } = usePendingStore();
  const { sources } = useSourceCRUD();

  const totalSelectedSources = useMemo(() => {
    let num = 0;

    Object.values(configuredSources).forEach((selectedSources) => {
      num += selectedSources.length;
    });

    return num;
  }, [configuredSources]);

  const renderActions = () => {
    if (!isSources || !sources.length) return null;

    const onSelect = (bool: boolean) => {
      if (bool) {
        const payload: Record<string, K8sActualSource[]> = {};

        sources.forEach((source) => {
          const id = { namespace: source.namespace, name: source.name, kind: source.kind };
          const isPending = isThisPending({ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: id });

          if (!isPending) {
            if (!payload[source.namespace]) {
              payload[source.namespace] = [source];
            } else {
              payload[source.namespace].push(source);
            }
          }
        });

        setConfiguredSources(payload);
      } else {
        setConfiguredSources({});
      }
    };

    return (
      <ActionsWrapper>
        <Checkbox value={sources.length === totalSelectedSources} onChange={onSelect} />
      </ActionsWrapper>
    );
  };

  return (
    <Container $nodeWidth={nodeWidth} className='nowheel nodrag'>
      {Icon && <Icon />}
      <Title size={14}>{title}</Title>
      <Badge label={tagValue} />
      {isSources && sourcesFetching && <FadeLoader />}

      {renderActions()}
    </Container>
  );
};

export default HeaderNode;
