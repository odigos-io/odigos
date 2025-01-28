import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const ErrorRoundIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.error;

  return (
    <svg width={size * (14 / 15)} height={size} viewBox='0 0 14 15' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M7 8.33673V6.00339M7 10.3055V10.3061M12.25 7.97266C12.25 10.8722 9.89949 13.2227 7 13.2227C4.1005 13.2227 1.75 10.8721 1.75 7.97265C1.75 5.07316 4.10051 2.72266 7 2.72266C9.8995 2.72266 12.25 5.07316 12.25 7.97266Z'
      />
    </svg>
  );
};
