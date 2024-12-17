import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const MetricsIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size * (16 / 17)} height={size} viewBox='0 0 16 17' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M13.2389 9.1672C13.2389 12.4305 10.5935 15.0758 7.33025 15.0758V14.3373M13.2389 9.1672H14.0883C14.0883 5.43485 11.0626 2.40918 7.33025 2.40918V4.73571M13.2389 9.1672L7.33025 9.1672M7.33025 14.3373C4.47491 14.3373 2.16016 12.0225 2.16016 9.1672H2.89877M7.33025 14.3373L7.33025 9.1672M2.89877 9.1672C2.89877 6.71975 4.88281 4.73571 7.33025 4.73571M2.89877 9.1672H7.33025M7.33025 4.73571L7.33025 9.1672'
      />
    </svg>
  );
};
