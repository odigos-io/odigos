import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const CodeAttributesIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeWidth='1.5'
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8 4C6.34315 4 5 5.19391 5 6.66667V9.33333C5 10.8061 3.65685 12 2 12C3.65685 12 5 13.1939 5 14.6667V17.3333C5 18.8061 6.34315 20 8 20M16 4C17.6569 4 19 5.19391 19 6.66667V9.33333C19 10.8061 20.3431 12 22 12C20.3431 12 19 13.1939 19 14.6667V17.3333C19 18.8061 17.6569 20 16 20M12.001 12H12M15.001 12H15M9.001 12H9'
      />
    </svg>
  );
};
