import React from 'react';
import { SVG } from '@/assets';

export const DeleteAttributeIcon: SVG = ({ size = 16, fill = '#B8B8B8', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M10.6666 10.0007L8.66659 8.00065M8.66659 8.00065L6.66659 6.00065M8.66659 8.00065L10.6666 6.00065M8.66659 8.00065L6.66659 10.0007M4.41666 4.01431C3.33136 5.06369 2.34376 6.24491 1.47059 7.5384C1.37903 7.67404 1.33325 7.83734 1.33325 8.00065C1.33325 8.16396 1.37903 8.32726 1.47059 8.4629C2.34376 9.75639 3.33136 10.9376 4.41666 11.987C4.65522 12.2177 4.77449 12.333 4.94023 12.4319C5.07791 12.514 5.25728 12.5866 5.41337 12.6232C5.60126 12.6673 5.78723 12.6673 6.15917 12.6673H11.3332C12.2667 12.6673 12.7334 12.6673 13.0899 12.4857C13.4035 12.3259 13.6585 12.0709 13.8183 11.7573C13.9999 11.4008 13.9999 10.9341 13.9999 10.0007V6.00065C13.9999 5.06723 13.9999 4.60052 13.8183 4.244C13.6585 3.9304 13.4035 3.67543 13.0899 3.51564C12.7334 3.33398 12.2667 3.33398 11.3332 3.33398H6.15917C5.78723 3.33398 5.60126 3.33398 5.41337 3.37809C5.25728 3.41473 5.07791 3.48727 4.94023 3.56943C4.77449 3.66832 4.65522 3.78365 4.41666 4.01431Z'
      />
    </svg>
  );
};
