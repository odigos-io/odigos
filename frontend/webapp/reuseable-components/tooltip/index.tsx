import React, { useState, useRef, ReactNode, useEffect } from 'react';
import { Text } from '../text';
import ReactDOM from 'react-dom';
import styled from 'styled-components';

interface TooltipProps {
  text: ReactNode;
  children: ReactNode;
}

const TooltipWrapper = styled.div`
  display: flex;
  position: relative;
  align-items: center;
`;

const TooltipContent = styled.div<{ $top: number; $left: number }>`
  position: absolute;
  top: ${({ $top }) => $top}px;
  left: ${({ $left }) => $left}px;
  transform: translateY(-100%);
  border-radius: 32px;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border: 1px solid ${({ theme }) => theme.colors.border};
  color: ${({ theme }) => theme.text.primary};
  padding: 16px;
  z-index: 9999;
  pointer-events: none;
  max-width: 300px;
`;

const Tooltip: React.FC<TooltipProps> = ({ text, children }) => {
  const [isHovered, setIsHovered] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const wrapperRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (wrapperRef.current) {
        const { top, left } = wrapperRef.current.getBoundingClientRect();

        setPosition({
          top: top + window.scrollY - 10, // Adjust the offset for the tooltip to be above the element
          left: left + window.scrollX,
        });
      }
    };

    if (isHovered) {
      document.addEventListener('mousemove', handleMouseMove);
    } else {
      document.removeEventListener('mousemove', handleMouseMove);
    }

    return () => document.removeEventListener('mousemove', handleMouseMove);
  }, [isHovered]);

  const tooltipContent = (
    <TooltipContent $top={position.top} $left={position.left}>
      <Text size={14}>{text}</Text>
    </TooltipContent>
  );

  if (text === '') {
    return <>{children}</>;
  }

  return (
    <TooltipWrapper ref={wrapperRef} onMouseEnter={() => setIsHovered(true)} onMouseLeave={() => setIsHovered(false)}>
      {children}
      {isHovered && ReactDOM.createPortal(tooltipContent, document.body)}
    </TooltipWrapper>
  );
};

export { Tooltip };
