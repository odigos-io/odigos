import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const ListIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M2.667 6.001h10.666M2.666 9.335h10.667M2.667 12.668H8m-5.333-10h10.666' />
    </svg>
  );
};
