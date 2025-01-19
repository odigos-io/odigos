import React from 'react';
import Lottie from 'react-lottie';
import styled from 'styled-components';
import animationData from './lottie.json';

interface Props {
  width?: number;
}

const Container = styled.div<{ $width: number }>`
  width: ${({ $width }) => $width / (620 / 220)}px;
  height: ${({ $width }) => $width}px;
  transform: rotate(-90deg);
`;

export const TraceLoader: React.FC<Props> = ({ width = 620 }) => {
  return (
    <Container $width={width}>
      <Lottie
        options={{
          loop: true,
          autoplay: true,
          animationData: animationData,
          rendererSettings: {
            preserveAspectRatio: 'xMidYMid slice',
          },
        }}
        height='100%'
        width='100%'
      />
    </Container>
  );
};
