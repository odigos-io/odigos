import React from 'react';
import { SVG } from '@/assets';

export const LogsIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size * (16 / 17)} height={size} viewBox='0 0 16 17' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M2.66699 8.5H6.66699M2.66699 12.5H6.66699M2.66699 4.5L13.3337 4.5M12.4765 12.0146C13.0334 11.61 13.5322 11.1357 13.96 10.6043C14.0138 10.5375 14.0138 10.4441 13.96 10.3773C13.5322 9.84585 13.0334 9.37156 12.4765 8.96696M10.1908 8.96696C9.63389 9.37156 9.13508 9.84585 8.7073 10.3773C8.65356 10.4441 8.65356 10.5375 8.7073 10.6043C9.13508 11.1357 9.63389 11.61 10.1908 12.0146'
      />
    </svg>
  );
};
