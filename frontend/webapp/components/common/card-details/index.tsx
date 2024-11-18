import React from 'react';
import styled from 'styled-components';
import { ConfiguredFields } from '@/components';
import { Text } from '@/reuseable-components';
interface CardDetailsProps {
  title?: string;
  data: {
    title: string;
    tooltip?: string;
    value: string;
  }[];
}

const CardDetails: React.FC<CardDetailsProps> = ({ data, title = 'Details' }) => {
  return (
    <Container>
      <TitleWrapper>
        <Text>{title}</Text>
      </TitleWrapper>
      <ConfiguredFields details={data} />
    </Container>
  );
};

export { CardDetails };

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
