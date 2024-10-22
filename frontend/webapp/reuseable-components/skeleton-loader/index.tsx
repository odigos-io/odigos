import React from 'react';
import styled, { keyframes } from 'styled-components';

const shimmer = keyframes`
  0% {
    background-position: -1000px 0;
  }
  100% {
    background-position: 1000px 0;
  }
`;

const SkeletonLoaderWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 1rem;
`;

const SkeletonItem = styled.div`
  display: flex;
  align-items: center;
  gap: 1rem;
`;

const SkeletonThumbnail = styled.div`
  width: 50px;
  height: 50px;
  border-radius: 8px;
  background: ${({ theme }) =>
    `linear-gradient(90deg, ${theme.colors.primary} 25%, ${theme.colors.primary} 50%, ${theme.colors.dark_grey} 75%)`};
  background-size: 200% 100%;
  animation: ${shimmer} 10s infinite linear;
`;

const SkeletonText = styled.div`
  flex: 1;
`;

const SkeletonLine = styled.div<{ width: string }>`
  height: 16px;
  margin-bottom: 0.5rem;
  background: ${({ theme }) =>
    `linear-gradient(90deg, ${theme.colors.primary} 25%, ${theme.colors.primary} 50%, ${theme.colors.dark_grey} 75%)`};
  background-size: 200% 100%;
  animation: ${shimmer} 1.5s infinite linear;
  width: ${(props) => props.width};
  border-radius: 4px;
`;

const SkeletonLoader: React.FC<{ size: number }> = ({ size = 5 }) => {
  return (
    <SkeletonLoaderWrapper>
      {[...Array(size)].map((_, index) => (
        <SkeletonItem key={index}>
          <SkeletonThumbnail />
          <SkeletonText>
            <SkeletonLine width="80%" />
            <SkeletonLine width="100%" />
          </SkeletonText>
        </SkeletonItem>
      ))}
    </SkeletonLoaderWrapper>
  );
};

export { SkeletonLoader };
