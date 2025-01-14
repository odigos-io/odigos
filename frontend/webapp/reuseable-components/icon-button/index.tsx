import React, { CSSProperties, PropsWithChildren } from 'react';
import { Tooltip } from '../tooltip';
import styled, { keyframes } from 'styled-components';

interface Props extends PropsWithChildren {
  onClick?: () => void;
  tooltip?: string;
  size?: number;
  withPing?: boolean;
  pingColor?: CSSProperties['backgroundColor'];
}

const Button = styled.button<{ $size: number }>`
  position: relative;
  width: ${({ $size }) => $size}px;
  height: ${({ $size }) => $size}px;
  border: none;
  border-radius: 100%;
  background-color: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  &:hover {
    background-color: ${({ theme }) => theme.colors.dropdown_bg_2};
  }
`;

const pingAnimation = keyframes`
  0% {
    transform: scale(1);
    opacity: 1;
  }
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
`;

const Ping = styled.div<{ $size: number; $color: Props['pingColor'] }>`
  position: absolute;
  top: ${({ $size }) => $size / 5}px;
  right: ${({ $size }) => $size / 5}px;
  width: 6px;
  height: 6px;
  border-radius: 100%;
  background-color: ${({ theme, $color }) => $color || theme.colors.secondary};

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 100%;
    background-color: ${({ theme, $color }) => $color || theme.colors.secondary};
    animation: ${pingAnimation} 1.5s cubic-bezier(0, 0, 0.2, 1) infinite;
  }
`;

export const IconButton: React.FC<Props> = ({ children, onClick, tooltip, size = 36, withPing, pingColor, ...props }) => {
  return (
    <Tooltip text={tooltip}>
      <Button $size={size} onClick={onClick} {...props}>
        {withPing && <Ping $size={size} $color={pingColor} />}
        {children}
      </Button>
    </Tooltip>
  );
};
