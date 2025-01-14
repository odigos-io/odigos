import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const DestinationsIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.info;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M7.9999 14.0999C11.3688 14.0999 14.0999 11.3688 14.0999 7.9999C14.0999 4.63096 11.3688 1.8999 7.9999 1.8999C4.63096 1.8999 1.8999 4.63096 1.8999 7.9999C1.8999 11.3688 4.63096 14.0999 7.9999 14.0999Z'
      />
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M7.99984 11.3332C9.84077 11.3332 11.3332 9.84077 11.3332 7.99984C11.3332 6.15889 9.84077 4.6665 7.99984 4.6665C6.15889 4.6665 4.6665 6.15889 4.6665 7.99984C4.6665 9.84077 6.15889 11.3332 7.99984 11.3332Z'
      />
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8.6665 7.99984C8.6665 8.36804 8.36804 8.6665 7.99984 8.6665C7.63164 8.6665 7.33317 8.36804 7.33317 7.99984C7.33317 7.63164 7.63164 7.33317 7.99984 7.33317C8.36804 7.33317 8.6665 7.63164 8.6665 7.99984Z'
      />
    </svg>
  );
};
