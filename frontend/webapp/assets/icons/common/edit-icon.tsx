import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const EditIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8 14C10.6787 11.8171 10.7261 16.2383 14 12.6667M2 13.997L3.81777 13.9999C4.07739 14.0003 4.2072 14.0005 4.32937 13.9712C4.43769 13.9452 4.54125 13.9022 4.63623 13.8438C4.74337 13.778 4.83516 13.6858 5.01874 13.5014L13.6676 4.81451C14.021 4.4596 14.1088 3.91087 13.8396 3.47722C13.5142 2.95298 13.0691 2.50221 12.5511 2.16754C12.136 1.8993 11.5908 1.95805 11.2417 2.30864L2.53993 11.0487C2.36296 11.2264 2.27447 11.3153 2.21029 11.4188C2.15338 11.5105 2.11072 11.6104 2.08378 11.7151C2.05341 11.8331 2.05031 11.9588 2.04411 12.2101L2 13.997Z'
      />
    </svg>
  );
};
