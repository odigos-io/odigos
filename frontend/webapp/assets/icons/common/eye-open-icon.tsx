import React from 'react';
import { SVG } from '@/assets';

export const EyeOpenIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M12 16.01C14.2091 16.01 16 14.2191 16 12.01C16 9.80087 14.2091 8.01001 12 8.01001C9.79086 8.01001 8 9.80087 8 12.01C8 14.2191 9.79086 16.01 12 16.01Z'
      />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M2 11.98C8.09 1.31996 15.91 1.32996 22 11.98' />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M22 12.01C15.91 22.67 8.09 22.66 2 12.01' />
    </svg>
  );
};
