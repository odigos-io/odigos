'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import animationData from './lottie.json';

// Must use dynamic, because the `react-lottie` package depends on `lottie-web`, and the `lottie-web` library directly interacts with the DOM;
// It requires browser-specific objects like document or window. This makes it incompatible with SSR.
const Lottie = dynamic(() => import('react-lottie'), { ssr: false });

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
