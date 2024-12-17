import React from 'react';
import { SVG } from '@/assets';

export const ArrowIcon: SVG = ({ size = 16, fill = '#F9F9F9', rotate = 0, onClick }) => {
  return (
    <svg width={size * (9 / 13.5)} height={size} viewBox='0 0 9 13.5' xmlns='http://www.w3.org/2000/svg' fill={fill} style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path d='M0.616717 8.03169L0.616717 5.70699L16.1519 5.70699L16.1519 8.03169L0.616717 8.03169ZM8.11144 -2.81613L9.7262 -1.10502L1.45534 6.87054L9.7262 14.9097L8.17631 16.6208L-1.19268 7.5802L-1.19268 6.1921L8.11144 -2.81613Z' />
    </svg>
  );
};
