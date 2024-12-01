import React from 'react';
import styled from 'styled-components';

interface Props extends React.PropsWithChildren {
  direction?: 'row' | 'column';
  size?: number;
}

const Container = styled.div<{ $direction: Props['direction']; $size: Props['size'] }>`
  display: flex;
  flex-direction: ${({ $direction }) => $direction};
  align-items: ${({ $direction }) => ($direction === 'row' ? 'center' : 'flex-start')};
  gap: ${({ $size }) => $size}px;
`;

export const Gap: React.FC<Props> = ({ direction = 'row', size = 4, children }) => {
  return (
    <Container $direction={direction} $size={size}>
      {children}
    </Container>
  );
};
