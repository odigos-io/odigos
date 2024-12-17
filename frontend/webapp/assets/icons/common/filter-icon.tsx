import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const FilterIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M5.33341 8H10.6667M7.33341 12H8.66675M2.66675 4H13.3334' />
    </svg>
  );
};
