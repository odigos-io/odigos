import React from 'react';
import Lottie from 'react-lottie';
import styled from 'styled-components';
import animationData from './lottie.json';

interface Props {
  width?: number;
}

const Container = styled.div<{ $width: number; $height: number }>`
  width: ${({ $width }) => $width}px;
  height: ${({ $height }) => $height}px;
  position: relative;
`;

export const TraceLoader: React.FC<Props> = ({ width: w = 620 }) => {
  const ratio = 620 / 220; // preserve aspect ratio
  const width = w / ratio;
  const height = w;

  return (
    // Note: The container width and height are swapped because the animation is rotated
    <Container $width={height} $height={width}>
      <Lottie
        width={width}
        height={height}
        isClickToPauseDisabled
        options={{
          loop: true,
          autoplay: true,
          animationData: animationData,
        }}
        style={{
          transform: 'rotate(-90deg)',
          position: 'absolute',
          top: -(width - width / 10),
          left: width - width / 10,
        }}
      />
    </Container>
  );
};
