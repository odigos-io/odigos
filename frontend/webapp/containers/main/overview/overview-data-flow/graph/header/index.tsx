'use client';
import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';

interface Column {
  icon: string;
  title: string;
  tagValue: number;
}

interface DataFlowHeaderProps {
  columns: Column[];
}

const HeaderContainer = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 0 32px;
  width: calc(100% - 64px);
`;

const ColumnContainer = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const TagValueContainer = styled.div`
  border: 1px solid ${theme.colors.border};
  padding: 0px 8px;
  border-radius: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 24px;
`;

const TagText = styled(Text)`
  font-family: ${({ theme }) => theme.font_family.secondary};
  color: ${({ theme }) => theme.text.grey};
`;

export const DataFlowHeader: React.FC<DataFlowHeaderProps> = ({ columns }) => (
  <HeaderContainer>
    {columns.map((column, index) => (
      <ColumnContainer key={index}>
        <Image src={column.icon} width={16} height={16} alt={column.title} />
        <Text size={14}>{column.title}</Text>
        <TagValueContainer>
          <TagText size={12}>{column.tagValue}</TagText>
        </TagValueContainer>
      </ColumnContainer>
    ))}
  </HeaderContainer>
);
