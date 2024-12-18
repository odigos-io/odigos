import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const SearchIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size * (14 / 15)} height={size} viewBox='0 0 14 15' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M12.25 13.165L10.2144 11.1294M10.2144 11.1294C11.1117 10.2322 11.6667 8.99258 11.6667 7.62337C11.6667 4.88496 9.44674 2.66504 6.70833 2.66504C3.96992 2.66504 1.75 4.88496 1.75 7.62337C1.75 10.3618 3.96992 12.5817 6.70833 12.5817C8.07754 12.5817 9.31712 12.0267 10.2144 11.1294Z'
      />
    </svg>
  );
};
