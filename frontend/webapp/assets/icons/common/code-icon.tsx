import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const CodeIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M5.33333 2.66669C4.22876 2.66669 3.33333 3.46263 3.33333 4.44446V6.22224C3.33333 7.20408 2.4379 8.00002 1.33333 8.00002C2.4379 8.00002 3.33333 8.79596 3.33333 9.7778V11.5556C3.33333 12.5374 4.22876 13.3334 5.33333 13.3334M10.6667 2.66669C11.7712 2.66669 12.6667 3.46263 12.6667 4.44446V6.22224C12.6667 7.20408 13.5621 8.00002 14.6667 8.00002C13.5621 8.00002 12.6667 8.79596 12.6667 9.7778V11.5556C12.6667 12.5374 11.7712 13.3334 10.6667 13.3334'
      />
    </svg>
  );
};
