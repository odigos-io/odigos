import React from 'react';
import { SVG } from '@/assets';

export const WarningTriangleIcon: SVG = ({ size = 16, fill = '#E9CF35', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8 8.66673V6.00006M8 10.9167V10.9175M7.07337 2.18915C7.66595 1.93695 8.33405 1.93695 8.92662 2.18915C10.6942 2.94145 14.8697 9.61453 14.7474 11.3981C14.6994 12.0972 14.3529 12.7408 13.7982 13.1614C12.323 14.2795 3.67698 14.2795 2.20185 13.1614C1.64705 12.7408 1.3006 12.0972 1.25263 11.3981C1.13026 9.61453 5.30575 2.94145 7.07337 2.18915Z'
      />
    </svg>
  );
};
