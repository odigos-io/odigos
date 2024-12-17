import React from 'react';
import { SVG } from '@/assets';

export const NoDataIcon: SVG = ({ size = 16, fill = '#7A7A7A', rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M6.31647 10.9497L7.96639 9.29983M7.96639 9.29983L9.61631 7.64992M7.96639 9.29983L6.31647 7.64992M7.96639 9.29983L9.61631 10.9497M14.6666 8.26667V9.73333C14.6666 11.2268 14.6666 11.9735 14.3759 12.544C14.1203 13.0457 13.7123 13.4537 13.2106 13.7094C12.6401 14 11.8934 14 10.3999 14H5.59992C4.10645 14 3.35971 14 2.78928 13.7094C2.28751 13.4537 1.87956 13.0457 1.6239 12.544C1.33325 11.9735 1.33325 11.2268 1.33325 9.73333V6.26667C1.33325 4.77319 1.33325 4.02646 1.6239 3.45603C1.87956 2.95426 2.28751 2.54631 2.78928 2.29065C3.35971 2 4.10645 2 5.59992 2H5.81029C6.12336 2 6.2799 2 6.42199 2.04315C6.54778 2.08135 6.6648 2.14398 6.76636 2.22745C6.88108 2.32174 6.96791 2.45199 7.14157 2.71248L7.52493 3.28752C7.69859 3.54801 7.78542 3.67826 7.90014 3.77255C8.0017 3.85602 8.11873 3.91865 8.24452 3.95685C8.38661 4 8.54314 4 8.85621 4H10.3999C11.8934 4 12.6401 4 13.2106 4.29065C13.7123 4.54631 14.1203 4.95426 14.3759 5.45603C14.6666 6.02646 14.6666 6.77319 14.6666 8.26667Z'
      />
    </svg>
  );
};
