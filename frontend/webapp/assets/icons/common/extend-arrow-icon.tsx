import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const ExtendArrowIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 12 12' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M4 5.06934C4.53103 5.80028 5.15354 6.45498 5.85106 7.01644C5.93869 7.08697 6.06131 7.08697 6.14894 7.01644C6.84646 6.45498 7.46897 5.80028 8 5.06934'
      />
    </svg>
  );
};
