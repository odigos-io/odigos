import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const PlusIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M7.99992 12.6663V7.99967M7.99992 7.99967V3.33301M7.99992 7.99967L3.33325 7.99967M7.99992 7.99967L12.6666 7.99967' />
    </svg>
  );
};
