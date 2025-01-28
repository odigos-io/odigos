import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const SunIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8 15.333v-.666m-5.185-1.482.471-.471M.666 8h.667m1.482-5.185.471.47M8 1.334V.667m4.714 2.619.471-.471M14.667 8h.666m-2.619 4.714.471.471M12 8a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z'
      />
    </svg>
  );
};
