import React from 'react';
import ReactDOM from 'react-dom';
import { useKeyDown } from '@/hooks';
import styled from 'styled-components';
import { fade, Overlay } from '@/styles';

interface DrawerProps {
  isOpen: boolean;
  onClose: () => void;
  closeOnEscape?: boolean;
  position?: 'right' | 'left';
  width?: string;
  children: React.ReactNode;
}

// Styled-component for drawer container
const DrawerContainer = styled.div<{
  isOpen: DrawerProps['isOpen'];
  position: DrawerProps['position'];
  width: DrawerProps['width'];
}>`
  position: fixed;
  top: 0;
  bottom: 0;
  ${({ position }) => position}: 0;
  z-index: 1000;
  width: ${({ width }) => width};
  background: ${({ theme }) => theme.colors.translucent_bg};
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
  overflow-y: auto;
  animation: ${({ isOpen, position = 'right' }) => (isOpen ? fade.in[position] : fade.out[position])} 0.3s ease;
`;

export const Drawer: React.FC<DrawerProps> = ({ isOpen, onClose, position = 'right', width = '300px', children, closeOnEscape = true }) => {
  useKeyDown(
    {
      key: 'Escape',
      active: isOpen && closeOnEscape,
    },
    () => onClose()
  );

  if (!isOpen) return null;

  return ReactDOM.createPortal(
    <>
      <Overlay hidden={!isOpen} onClick={onClose} />
      <DrawerContainer isOpen={isOpen} position={position} width={width}>
        {children}
      </DrawerContainer>
    </>,
    document.body
  );
};
