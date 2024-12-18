import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const CheckCircledIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size * (16 / 17)} viewBox='0 0 17 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M6.41707 8.34197L7.97787 9.90111C8.72855 8.58846 9.76744 7.46337 11.0162 6.61065L11.0837 6.56453M14.8504 8.00039C14.8504 11.3693 12.1193 14.1004 8.75039 14.1004C5.38145 14.1004 2.65039 11.3693 2.65039 8.00039C2.65039 4.63145 5.38145 1.90039 8.75039 1.90039C12.1193 1.90039 14.8504 4.63145 14.8504 8.00039Z'
      />
    </svg>
  );
};
