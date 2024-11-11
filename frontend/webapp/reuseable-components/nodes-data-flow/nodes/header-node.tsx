'use client';
import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { Badge, Text } from '@/reuseable-components';

const ColumnContainer = styled.div<{ nodeWidth: number }>`
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

interface Column {
  icon: string;
  title: string;
  tagValue: number;
}

interface HeaderNodeProps {
  data: Column;
  nodeWidth: number;
}

const HeaderNode = ({ data, nodeWidth }: HeaderNodeProps) => {
  return (
    <ColumnContainer nodeWidth={nodeWidth}>
      <Image src={data.icon} width={16} height={16} alt={data.title} />
      <Title size={14}>{data.title}</Title>
      <Badge label={data.tagValue} />
    </ColumnContainer>
  );
};

export default HeaderNode;
