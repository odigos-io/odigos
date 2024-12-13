import React, { CSSProperties, PropsWithChildren } from 'react';
import styled, { keyframes } from 'styled-components';

interface Props extends PropsWithChildren {
  onClick?: () => void;
  withPing?: boolean;
  pingColor?: CSSProperties['backgroundColor'];
}

const Button = styled.button`
  position: relative;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 100%;
  background-color: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  &:hover {
    background-color: ${({ theme }) => theme.colors.white_opacity['008']};
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

const Ping = styled.div<{ $color: Props['pingColor'] }>`
  position: absolute;
  top: 8px;
  right: 8px;
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

export const IconButton: React.FC<Props> = ({ children, onClick, withPing, pingColor }) => {
  return (
    <Button onClick={onClick}>
      {withPing && <Ping $color={pingColor} />}
      {children}
    </Button>
  );
};
