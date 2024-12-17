import React from 'react';
import { SVG } from '@/assets';

export const CrossCircledIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size * (16 / 17)} viewBox='0 0 17 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M6.75049 10.0005L8.75049 8.00049M8.75049 8.00049L10.7505 6.00049M8.75049 8.00049L6.75049 6.00049M8.75049 8.00049L10.7505 10.0005M8.75039 14.1004C5.38145 14.1004 2.65039 11.3693 2.65039 8.00039C2.65039 4.63145 5.38145 1.90039 8.75039 1.90039C12.1193 1.90039 14.8504 4.63145 14.8504 8.00039C14.8504 11.3693 12.1193 14.1004 8.75039 14.1004Z'
      />
    </svg>
  );
};
