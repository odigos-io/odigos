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
  background: #1a1a1a;
  color: #fff;
  text-align: center;
  border-radius: 4px;
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
    margin-left: -5px;
    border-width: 5px;
    border-style: solid;
    border-color: #1a1a1a transparent transparent transparent;
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
            <Text>{text}</Text>
          </TooltipText>
        )}
      </TooltipWrapper>
    </TooltipContainer>
  );
};

export { Tooltip };
