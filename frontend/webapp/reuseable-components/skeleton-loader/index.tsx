import React from 'react';
import { FlexColumn } from '@/styles';
import styled, { keyframes } from 'styled-components';

const shimmer = keyframes<{ $width: string }>`
  0% {
    background-position: -500px 0;
  }
  100% {
    background-position: 500px 0;
  }
`;

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 16px;
`;

const SkeletonItem = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
`;

const Thumbnail = styled.div`
  width: 50px;
  height: 50px;
  border-radius: 8px;
  background: ${({ theme }) => `linear-gradient(90deg, ${theme.colors.dropdown_bg_2} 25%, ${theme.colors.dropdown_bg_2} 50%, ${theme.colors.border} 75%)`};
  background-size: 200% 100%;
  animation: ${shimmer} 10s infinite linear;
`;

const LineWrapper = styled(FlexColumn)`
  flex: 1;
  gap: 12px;
`;

const Line = styled.div<{ $width: string }>`
  width: ${({ $width }) => $width};
  height: 16px;
  background: ${({ theme }) => `linear-gradient(90deg, ${theme.colors.dropdown_bg_2} 25%, ${theme.colors.dropdown_bg_2} 50%, ${theme.colors.border} 75%)`};
  background-size: 200% 100%;
  animation: ${shimmer} 1.5s infinite linear;
  border-radius: 4px;
`;

export const SkeletonLoader: React.FC<{ size?: number }> = ({ size = 5 }) => {
  return (
    <Container>
      {[...Array(size)].map((_, index) => (
        <SkeletonItem key={index}>
          <Thumbnail />
          <LineWrapper>
            <Line $width='80%' />
            <Line $width='100%' />
          </LineWrapper>
        </SkeletonItem>
      ))}
    </Container>
  );
};
