import React from 'react';
import ReactDOM from 'react-dom';
import styled from 'styled-components';
import { slide, Overlay } from '@/styles';
import { useKeyDown, useTransition } from '@/hooks';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  closeOnEscape?: boolean;
  position?: 'right' | 'left';
  width?: string;
  children: React.ReactNode;
}

const Container = styled.div<{
  $position: Props['position'];
  $width: Props['width'];
}>`
  position: fixed;
  top: 0;
  bottom: 0;
  ${({ $position }) => $position}: 0;
  z-index: 1000;
  width: ${({ $width }) => $width};
  background: ${({ theme }) => theme.colors.translucent_bg};
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
  overflow-y: auto;
`;

export const Drawer: React.FC<Props> = ({ isOpen, onClose, position = 'right', width = '300px', children, closeOnEscape = true }) => {
  useKeyDown({ key: 'Escape', active: isOpen && closeOnEscape }, () => onClose());

  const Transition = useTransition({
    container: Container,
    animateIn: slide.in[position],
    animateOut: slide.out[position],
  });

  if (!isOpen) return null;

  return ReactDOM.createPortal(
    <>
      <Overlay hidden={!isOpen} onClick={onClose} />

      <Transition enter={isOpen} $position={position} $width={width}>
        {children}
      </Transition>
    </>,
    document.body,
  );
};
