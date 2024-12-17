import React from 'react';
import { SVG } from '@/assets';

export const TrashIcon: SVG = ({ size = 16, fill = '#E25A5A', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M10.6665 4.00065L9.92946 2.52655C9.56401 1.79567 8.81699 1.33398 7.99984 1.33398C7.18268 1.33398 6.43566 1.79567 6.07022 2.52655L5.33317 4.00065M2.6665 4.00065H13.3332M3.99984 4.00065H11.9998V10.0007C11.9998 11.2432 11.9998 11.8644 11.7968 12.3545C11.5262 13.0079 11.0071 13.527 10.3537 13.7977C9.8636 14.0007 9.24235 14.0007 7.99984 14.0007C6.75733 14.0007 6.13607 14.0007 5.64601 13.7977C4.99261 13.527 4.47348 13.0079 4.20283 12.3545C3.99984 11.8644 3.99984 11.2432 3.99984 10.0007V4.00065Z'
      />
    </svg>
  );
};
