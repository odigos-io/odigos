import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const NotebookIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8.00016 3.37696V14.3327M8.00016 3.37696C10.0168 2.64365 12.6703 2.23608 14.6668 3.37696V13.7103C12.5356 12.7969 9.9581 13.4427 8.00016 14.3327M8.00016 3.37696C5.98356 2.64365 3.33004 2.23608 1.3335 3.37696V13.7103C3.46472 12.7969 6.04222 13.4427 8.00016 14.3327'
      />
    </svg>
  );
};
