import React from 'react';
import { SVG } from '@/assets';

export const RenameAttributeIcon: SVG = ({ size = 16, fill = '#B8B8B8' }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none'>
      <path
        stroke={fill}
        stroke-linecap='round'
        stroke-linejoin='round'
        d='M14.6666 8.00065V12.0007M1.33325 12.6673L4.23847 4.1055C4.40371 3.64283 4.84196 3.33398 5.33325 3.33398C5.83824 3.33398 6.28547 3.66 6.43999 4.14076L9.33325 12.6673M2.69054 8.66732H7.97596M14.6666 10.0007C14.6666 11.1052 13.7712 12.0007 12.6666 12.0007C11.562 12.0007 10.6666 11.1052 10.6666 10.0007C10.6666 8.89608 11.562 8.00065 12.6666 8.00065C13.7712 8.00065 14.6666 8.89608 14.6666 10.0007Z'
      />
    </svg>
  );
};
