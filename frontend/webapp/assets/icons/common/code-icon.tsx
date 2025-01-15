import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const CodeIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M11.334 12a18.802 18.802 0 0 0 3.231-3.66.62.62 0 0 0 0-.68A18.8 18.8 0 0 0 11.334 4m-6.665 8a18.803 18.803 0 0 1-3.231-3.66.62.62 0 0 1 0-.68A18.801 18.801 0 0 1 4.669 4m4.667-1.332L6.669 13.335'
      />
    </svg>
  );
};
