import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const AvatarIcon: SVG = ({ size = 16, rotate = 0, onClick }) => {
  return (
    <svg width={size * (28 / 29)} height={size} viewBox='0 0 28 29' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        fill={theme.text.secondary}
        fillOpacity='0.08'
        d='M0 14.5469C0 6.81489 6.26801 0.546875 14 0.546875C21.732 0.546875 28 6.81489 28 14.5469C28 22.2789 21.732 28.5469 14 28.5469C6.26801 28.5469 0 22.2789 0 14.5469Z'
      />
      <path
        stroke={theme.text.grey}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M16.6667 11.2135C16.6667 12.6863 15.4728 13.8802 14.0001 13.8802C12.5273 13.8802 11.3334 12.6863 11.3334 11.2135C11.3334 9.74078 12.5273 8.54688 14.0001 8.54688C15.4728 8.54688 16.6667 9.74078 16.6667 11.2135Z'
      />
      <path
        stroke={theme.text.grey}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M16.6667 16.5469H11.3334C9.86066 16.5469 8.66675 17.7408 8.66675 19.2135C8.66675 19.9499 9.2637 20.5469 10.0001 20.5469H18.0001C18.7365 20.5469 19.3334 19.9499 19.3334 19.2135C19.3334 17.7408 18.1395 16.5469 16.6667 16.5469Z'
      />
    </svg>
  );
};
