import React from 'react';
import styled from 'styled-components';
import { Text } from '../text';

interface TooltipProps {
  text: string;
  children: React.ReactNode;
}

const TooltipContainer = styled.div`
  position: relative;
  display: inline-block;
  width: fit-content;

  cursor: pointer;
`;

const TooltipText = styled.div`
  visibility: hidden;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border: 1px solid ${({ theme }) => theme.colors.border};
  color: ${({ theme }) => theme.text.primary};
  text-align: center;
  border-radius: 32px;
  padding: 8px;
  position: absolute;
  z-index: 1;
  bottom: 125%; /* Position the tooltip above the text */
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.3s;

  /* Tooltip arrow */
  &::after {
    content: '';
    position: absolute;
    z-index: 99999;
    top: 100%; /* At the bottom of the tooltip */
    left: 50%;
  }
`;

const TooltipWrapper = styled.div<{ hasText: boolean }>`
  &:hover ${TooltipText} {
    ${({ hasText }) =>
      hasText &&
      `
      visibility: visible;
      opacity: 1;
      z-index: 999;
    `}
  }
`;

const Tooltip: React.FC<TooltipProps> = ({ text, children }) => {
  const hasText = !!text;

  return (
    <TooltipContainer>
      <TooltipWrapper hasText={hasText}>
        {children}
        {hasText && (
          <TooltipText>
            <Text size={14}>{text}</Text>
          </TooltipText>
        )}
      </TooltipWrapper>
    </TooltipContainer>
  );
};

export { Tooltip };
