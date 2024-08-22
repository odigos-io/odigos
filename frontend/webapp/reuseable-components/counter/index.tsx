import React from 'react';
import styled from 'styled-components';
import { Text } from '../text';

interface CounterProps {
  value: number;
  title: string;
}

const Container = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const ValueContainer = styled.div<{ value?: number }>`
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 32px;
  border: 1px solid rgba(249, 249, 249, 0.24);
  background: ${({ value, theme }) =>
    value ? theme.colors.majestic_blue : 'transparent'};
`;

const Value = styled(Text)<{ value?: number }>`
  opacity: ${({ value }) => (value ? 1 : 0.8)};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-size: 12px;
`;

const Counter: React.FC<CounterProps> = ({ value, title }) => {
  return (
    <Container>
      <Text>{title}</Text>
      <ValueContainer value={value}>
        <Value>{value}</Value>
      </ValueContainer>
    </Container>
  );
};

export { Counter };
