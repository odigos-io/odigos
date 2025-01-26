import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const MoonIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M14 7.977A4.333 4.333 0 1 1 8.023 2H8a6 6 0 1 0 6 6v-.023Z' />
    </svg>
  );
};
