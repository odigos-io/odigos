import React from 'react';
import { SVG } from '@/assets';

export const CheckIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size * (16 / 17)} height={size} viewBox='0 0 16 17' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M3.6665 9.24833L6.67773 12.7315L6.94495 12.2642C8.37771 9.75889 10.3605 7.61153 12.7439 5.98403L13.3332 5.58167' />
    </svg>
  );
};
