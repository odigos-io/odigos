import React from 'react';
import { SVG } from '@/assets';

export const SamplerIcon: SVG = ({ size = 16, fill = '#B8B8B8' }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none'>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M5.67367 5.67497H5.687M10.515 3.91532L12.1293 5.52965C13.0067 6.40707 13.4454 6.84578 13.6851 7.31618C14.1695 8.2669 14.1695 9.39207 13.6851 10.3428C13.4454 10.8132 13.0067 11.2519 12.1293 12.1293C11.2519 13.0067 10.8132 13.4454 10.3428 13.6851C9.39207 14.1695 8.2669 14.1695 7.31618 13.6851C6.84578 13.4454 6.40707 13.0067 5.52965 12.1293L3.91532 10.515C3.17317 9.77284 2.8021 9.40176 2.54483 8.96668C2.31682 8.58108 2.15521 8.15991 2.06673 7.72076C1.9669 7.22527 1.99448 6.70122 2.04965 5.65312L2.07845 5.10584C2.13195 4.0893 2.1587 3.58104 2.36993 3.18719C2.55598 2.84028 2.84028 2.55598 3.18719 2.36993C3.58104 2.1587 4.0893 2.13195 5.10584 2.07845L5.65312 2.04965C6.70122 1.99448 7.22527 1.9669 7.72076 2.06673C8.15991 2.15521 8.58108 2.31682 8.96668 2.54483C9.40176 2.8021 9.77284 3.17317 10.515 3.91532ZM6.32633 5.65853C6.32633 6.02672 6.02786 6.3252 5.65967 6.3252C5.29148 6.3252 4.993 6.02672 4.993 5.65853C4.993 5.29034 5.29148 4.99186 5.65967 4.99186C6.02786 4.99186 6.32633 5.29034 6.32633 5.65853Z'
      />
    </svg>
  );
};
