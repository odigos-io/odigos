import React from 'react';
import { SVG } from '@/assets';

export const CrossIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M4 12L8 8M8 8L12 4M8 8L4 4M8 8L12 12' />
    </svg>
  );
};
