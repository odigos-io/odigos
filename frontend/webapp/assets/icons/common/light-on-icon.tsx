import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const LightOnIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M10.3789 21H13.621M12 2V1M19 4.70711L19.7071 4M4.70711 4.70711L4 4M22 11H21M3 11H2M6.3127 10.468C6.3127 7.44492 8.85908 4.99427 12.0002 4.99427C15.1413 4.99427 17.6877 7.44492 17.6877 10.468C17.6877 12.125 16.9226 13.6102 15.7138 14.6139C15.2032 15.0379 14.7642 15.5643 14.5974 16.2066L14.37 17.0819C14.2304 17.6192 13.7455 17.9943 13.1904 17.9943H10.81C10.2549 17.9943 9.76995 17.6192 9.63039 17.0819L9.40302 16.2066C9.23616 15.5643 8.7972 15.0379 8.28662 14.6139C7.07782 13.6102 6.3127 12.125 6.3127 10.468Z'
      />
    </svg>
  );
};
