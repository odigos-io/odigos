import React from 'react';
import { SVG } from '@/assets';

export const OverviewIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size * (14 / 15)} height={size} viewBox='0 0 14 15' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M1.75 8.79362H12.25M1.75 11.7103H12.25M3.5 5.87695H10.5C11.4665 5.87695 12.25 5.09345 12.25 4.12695C12.25 3.16045 11.4665 2.37695 10.5 2.37695H3.5C2.5335 2.37695 1.75 3.16045 1.75 4.12695C1.75 5.09345 2.5335 5.87695 3.5 5.87695Z'
      />
    </svg>
  );
};
