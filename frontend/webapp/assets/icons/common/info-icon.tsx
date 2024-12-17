import React from 'react';
import { SVG } from '@/assets';

export const InfoIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0 }) => {
  return (
    <svg width={size * (16 / 17)} height={size} viewBox='0 0 16 17' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8 8.91498V11.5816M8 6.66498V6.66423M14 8.91504C14 12.2287 11.3137 14.915 8 14.915C4.68629 14.915 2 12.2287 2 8.91504C2 5.60133 4.68629 2.91504 8 2.91504C11.3137 2.91504 14 5.60133 14 8.91504Z'
      />
    </svg>
  );
};
