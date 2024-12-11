import React, { useMemo } from 'react';
import Image from 'next/image';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { useSourceCRUD } from '@/hooks';
import type { Node, NodeProps } from '@xyflow/react';
import { Badge, Checkbox, Text } from '@/reuseable-components';

interface Props
  extends NodeProps<
    Node<
      {
        icon: string;
        title: string;
        tagValue: number;
      },
      'header'
    >
  > {
  nodeWidth: number;
}

const Container = styled.div<{ $nodeWidth: Props['nodeWidth'] }>`
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

const HeaderNode: React.FC<Props> = ({ nodeWidth, data }) => {
  const { title, icon, tagValue } = data;
  const isSources = title === 'Sources';
  const extraWidth = 40;

  const { configuredSources, setConfiguredSources } = useAppStore((state) => state);
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
        const payload = {};

        sources.forEach((source) => {
          if (!payload[source.namespace]) {
            payload[source.namespace] = [source];
          } else {
            payload[source.namespace].push(source);
          }
        });

        setConfiguredSources(payload);
      } else {
        setConfiguredSources({});
      }
    };

    return (
      <ActionsWrapper>
        <Checkbox initialValue={sources.length === totalSelectedSources} onChange={onSelect} />
      </ActionsWrapper>
    );
  };

  return (
    <Container $nodeWidth={nodeWidth + extraWidth}>
      <Image src={icon} width={16} height={16} alt={title} />
      <Title size={14}>{title}</Title>
      <Badge label={tagValue} />

      {renderActions()}
    </Container>
  );
};

export default HeaderNode;
