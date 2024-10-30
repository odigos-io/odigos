'use client';
import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';

const ColumnContainer = styled.div<{ columnWidth: number }>`
  width: ${({ columnWidth }) => `${columnWidth + 40}px`};
  padding: 12px 0px 16px 0px;
  gap: 8px;
  display: flex;
  align-items: center;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
`;

const TagValueContainer = styled.div`
  border: 1px solid ${({ theme }) => theme.colors.border};
  padding: 0px 8px;
  border-radius: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 24px;
`;

const Title = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
`;

const TagText = styled(Title)`
  font-family: ${({ theme }) => theme.font_family.secondary};
`;

interface Column {
  icon: string;
  title: string;
  tagValue: number;
}

interface HeaderNodeProps {
  data: Column;
  columnWidth: number;
}

const HeaderNode = ({ data, columnWidth }: HeaderNodeProps) => {
  return (
    <ColumnContainer columnWidth={columnWidth}>
      <Image src={data.icon} width={16} height={16} alt={data.title} />
      <Title size={14}>{data.title}</Title>
      <TagValueContainer>
        <TagText size={12}>{data.tagValue}</TagText>
      </TagValueContainer>
    </ColumnContainer>
  );
};

export default HeaderNode;
