import React from 'react';
import { SVG } from '@/assets';

export const SourcesIcon: SVG = ({ size = 16, fill = '#B8B8B8', rotate = 0 }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M13.3332 3.8V8M13.3332 3.8C13.3332 4.79411 10.9454 5.6 7.99984 5.6C5.05432 5.6 2.6665 4.79411 2.6665 3.8M13.3332 3.8C13.3332 2.80589 10.9454 2 7.99984 2C5.05432 2 2.6665 2.80589 2.6665 3.8M13.3332 8V12.0875C13.3332 13.1437 10.9454 14 7.99984 14C5.05432 14 2.6665 13.1437 2.6665 12.0875V8M13.3332 8C13.3332 8.99413 10.9454 9.8 7.99984 9.8C5.05432 9.8 2.6665 8.99413 2.6665 8M2.6665 8V3.8M10.6665 7.33333H10.6732M10.6665 11.3333H10.6732'
      />
    </svg>
  );
};
