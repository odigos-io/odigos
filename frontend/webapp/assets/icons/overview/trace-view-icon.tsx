import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const TraceViewIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size * (16 / 17)} height={size} viewBox='0 0 16 17' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M2.66666 8.61719H6.66666M2.66666 12.6172H6.66666M2.66666 4.61719H13.3333M14 12.9505L13.0809 12.0315M13.0809 12.0315C13.4438 11.6686 13.6667 11.1686 13.6667 10.6172C13.6667 9.51147 12.7724 8.61719 11.6667 8.61719C10.5609 8.61719 9.66666 9.51147 9.66666 10.6172C9.66666 11.7229 10.5609 12.6172 11.6667 12.6172C12.2181 12.6172 12.7181 12.3943 13.0809 12.0315Z'
      />
    </svg>
  );
};
