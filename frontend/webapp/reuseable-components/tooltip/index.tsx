import React, { useState, PropsWithChildren, useRef, MouseEvent, forwardRef } from 'react';
import ReactDOM from 'react-dom';
import { Text } from '..';
import { InfoIcon } from '@/assets';
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
  const popupRef = useRef<HTMLDivElement>(null);

  const handleMouseEvent = (e: MouseEvent) => {
    const { type, clientX, clientY } = e;
    const { innerWidth, innerHeight } = window;

    let top = clientY;
    let left = clientX;

    if (top >= innerHeight / 2) top += -(popupRef.current?.clientHeight || 40);
    if (left >= innerWidth / 2) left += -(popupRef.current?.clientWidth || Math.min((text?.length || 0) * 7.5, 300));

    setPopupPosition({ top, left });
    setIsHovered(type !== 'mouseleave');
  };

  if (!text) return <>{children}</>;

  return (
    <TooltipContainer onMouseEnter={handleMouseEvent} onMouseMove={handleMouseEvent} onMouseLeave={handleMouseEvent}>
      {children}
      {withIcon && <InfoIcon />}
      {isHovered && (
        <Popup ref={popupRef} {...popupPosition}>
          {text}
        </Popup>
      )}
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
  border: 1px solid ${({ theme }) => theme.colors.dropdown_bg_2};
  background-color: ${({ theme }) => theme.colors.info};
  color: ${({ theme }) => theme.text.primary};

  pointer-events: none;
`;

const Popup = forwardRef<HTMLDivElement, PopupProps>(({ top, left, children }, ref) => {
  return ReactDOM.createPortal(
    <PopupContainer ref={ref} $top={top} $left={left}>
      <Text size={12}>{children}</Text>
    </PopupContainer>,
    document.body,
  );
});
