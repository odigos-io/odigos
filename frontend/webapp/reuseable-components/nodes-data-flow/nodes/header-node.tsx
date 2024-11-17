import React, { useMemo } from 'react';
import Image from 'next/image';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { useSourceCRUD } from '@/hooks';
import { Badge, Checkbox, Text } from '@/reuseable-components';

interface Column {
  icon: string;
  title: string;
  tagValue: number;
}

interface HeaderNodeProps {
  nodeWidth: number;
  data: Column;
}

const Container = styled.div<{ nodeWidth: HeaderNodeProps['nodeWidth'] }>`
  width: ${({ nodeWidth }) => `${nodeWidth + 40}px`};
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
  margin-right: 24px;
`;

const HeaderNode = ({ data, nodeWidth }: HeaderNodeProps) => {
  const { title, icon, tagValue } = data;
  const { configuredSources, setConfiguredSources } = useAppStore((state) => state);
  const { sources } = useSourceCRUD();

  const totalSelected = useMemo(() => {
    let num = 0;
    if (title !== 'Sources') return num;

    Object.values(configuredSources).forEach((selectedSources) => {
      num += selectedSources.length;
    });

    return num;
  }, [title, configuredSources]);

  const sourcesToSelect = useMemo(() => {
    const payload = {};
    if (title !== 'Sources') return payload;

    sources.forEach((source) => {
      if (!payload[source.namespace]) {
        payload[source.namespace] = [source];
      } else {
        payload[source.namespace].push(source);
      }
    });

    return payload;
  }, [title, sources]);

  const renderActions = () => {
    if (title !== 'Sources') return null;

    const isDisabled = !sources.length;
    const isSelected = !isDisabled && sources.length === totalSelected;
    const onSelect = (bool: boolean) => setConfiguredSources(bool ? sourcesToSelect : {});

    return (
      <ActionsWrapper>
        <Checkbox disabled={isDisabled} initialValue={isSelected} onChange={onSelect} />
      </ActionsWrapper>
    );
  };

  return (
    <Container nodeWidth={nodeWidth}>
      <Image src={icon} width={16} height={16} alt={title} />
      <Title size={14}>{title}</Title>
      <Badge label={tagValue} />

      {renderActions()}
    </Container>
  );
};

export default HeaderNode;
