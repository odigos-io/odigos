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

const ValueContainer = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 32px;
  border: 1px solid rgba(249, 249, 249, 0.24);
`;

const Counter: React.FC<CounterProps> = ({ value, title }) => {
  return (
    <Container>
      <Text>{title}</Text>
      <ValueContainer>
        <Text size={12} opacity={0.8} family={'secondary'}>
          {value}
        </Text>
      </ValueContainer>
    </Container>
  );
};

export { Counter };
