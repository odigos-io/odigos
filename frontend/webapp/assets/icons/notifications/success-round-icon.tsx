import React from 'react';
import { SVG } from '@/assets';

export const SuccessRoundIcon: SVG = ({ size = 16, fill = '#81AF65' }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none'>
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='m5.667 8.341 1.56 1.56a9.99 9.99 0 0 1 3.039-3.29l.067-.047M14.1 8A6.1 6.1 0 1 1 1.9 8a6.1 6.1 0 0 1 12.2 0Z' />
    </svg>
  );
};
