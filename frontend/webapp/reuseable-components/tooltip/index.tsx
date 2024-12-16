import React, { useState, PropsWithChildren } from 'react';
import Image from 'next/image';
import ReactDOM from 'react-dom';
import { Text } from '../text';
import styled from 'styled-components';

interface Position {
  top: number;
  left: number;
}

interface TooltipProps extends PropsWithChildren {
  text?: string;
  withIcon?: boolean;
}

interface PopupProps extends PropsWithChildren, Position {}

const TooltipContainer = styled.div`
  position: relative;
  display: flex;
  align-items: center;
  gap: 4px;
`;

export const Tooltip: React.FC<TooltipProps> = ({ text, withIcon, children }) => {
  const [isHovered, setIsHovered] = useState(false);
  const [popupPosition, setPopupPosition] = useState<Position>({ top: 0, left: 0 });

  const handleMouseEvent = (e: React.MouseEvent) => {
    const { type, clientX, clientY } = e;
    const { innerWidth, innerHeight } = window;

    let top = clientY;
    let left = clientX;
    const textLen = text?.length || 0;

    if (top >= innerHeight / 2) top += -40;
    if (left >= innerWidth / 2) left += -(textLen * 8);

    setPopupPosition({ top, left });
    setIsHovered(type !== 'mouseleave');
  };

  if (!text) return <>{children}</>;

  return (
    <TooltipContainer onMouseEnter={handleMouseEvent} onMouseMove={handleMouseEvent} onMouseLeave={handleMouseEvent}>
      {children}
      {withIcon && <Image src='/icons/common/info.svg' alt='info' width={16} height={16} />}
      {isHovered && <Popup {...popupPosition}>{text}</Popup>}
    </TooltipContainer>
  );
};

const PopupContainer = styled.div<{ $top: number; $left: number }>`
  position: absolute;
  top: ${({ $top }) => $top}px;
  left: ${({ $left }) => $left}px;
  z-index: 9999;

  max-width: 270px;
  padding: 8px 12px;
  border-radius: 16px;
  border: 1px solid ${({ theme }) => theme.colors.white_opacity['008']};
  background-color: ${({ theme }) => theme.colors.info};
  color: ${({ theme }) => theme.text.primary};

  pointer-events: none;
`;

const Popup: React.FC<PopupProps> = ({ top, left, children }) => {
  return ReactDOM.createPortal(
    <PopupContainer $top={top} $left={left}>
      <Text size={12}>{children}</Text>
    </PopupContainer>,
    document.body,
  );
};
