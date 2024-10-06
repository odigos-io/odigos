import React, { useState, useEffect } from 'react';
import styled, { css } from 'styled-components';

interface DrawerProps {
  isOpen: boolean;
  onClose: () => void;
  position?: 'left' | 'right'; // Optional prop to specify the drawer opening side
  width?: string; // Optional width control, defaults to 300px
  children: React.ReactNode;
}

// Styled-component for overlay
const Overlay = styled.div<{ isOpen: boolean }>`
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  opacity: ${({ isOpen }) => (isOpen ? 1 : 0)};
  transition: opacity 0.3s ease;
  visibility: ${({ isOpen }) => (isOpen ? 'visible' : 'hidden')};
  z-index: 999;
`;

// Styled-component for drawer container
const DrawerContainer = styled.div<{
  isOpen: boolean;
  position: 'left' | 'right';
  width: string;
}>`
  position: fixed;
  top: 0;
  bottom: 0;
  ${({ position, width }) => position}: 0;
  width: ${({ width }) => width};
  background-color: ${({ theme }) => theme.colors.light_dark};
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
  transform: translateX(
    ${({ isOpen, position }) =>
      isOpen ? '0' : position === 'left' ? '-100%' : '100%'}
  );
  transition: transform 0.3s ease;
  z-index: 1000;
  overflow-y: auto;
`;

export const Drawer: React.FC<DrawerProps> = ({
  isOpen,
  onClose,
  position = 'right',
  width = '300px',
  children,
}) => {
  // Handle closing the drawer when escape key is pressed
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, onClose]);

  return (
    <>
      <Overlay isOpen={isOpen} onClick={onClose} />
      <DrawerContainer isOpen={isOpen} position={position} width={width}>
        {children}
      </DrawerContainer>
    </>
  );
};
