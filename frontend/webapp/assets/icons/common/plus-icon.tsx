import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const PlusIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M7.99992 12.6663V7.99967M7.99992 7.99967V3.33301M7.99992 7.99967L3.33325 7.99967M7.99992 7.99967L12.6666 7.99967' />
    </svg>
  );
};
