import React from 'react';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';
import { ConfiguredFields } from '@/components';

interface Props {
  title?: string;
  data: {
    title: string;
    tooltip?: string;
    value: string;
  }[];
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  padding: 16px 24px 24px 24px;
  flex-direction: column;
  align-items: flex-start;
  gap: 16px;
  align-self: stretch;
  border-radius: 24px;
  border: 1px solid ${({ theme }) => theme.colors.border};
`;

const TitleWrapper = styled.div``;

export const CardDetails: React.FC<Props> = ({ title = 'Details', data }) => {
  return (
    <Container>
      <TitleWrapper>
        <Text>{title}</Text>
      </TitleWrapper>
      <ConfiguredFields details={data} />
    </Container>
  );
};
