import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const CopyIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M11.2679 11.2679C11.425 11.2446 11.5649 11.213 11.6967 11.1702C12.7115 10.8405 13.5071 10.0449 13.8369 9.03006C14 8.52795 14 7.90752 14 6.66667C14 5.42581 14 4.80539 13.8369 4.30328C13.5071 3.28848 12.7115 2.49287 11.6967 2.16314C11.1946 2 10.5742 2 9.33333 2C8.09248 2 7.47205 2 6.96994 2.16314C5.95515 2.49287 5.15954 3.28848 4.82981 4.30328C4.787 4.43505 4.75542 4.57498 4.73212 4.73212M11.2679 11.2679C11.3333 10.8262 11.3333 10.2485 11.3333 9.33333C11.3333 8.09248 11.3333 7.47205 11.1702 6.96994C10.8405 5.95515 10.0449 5.15954 9.03006 4.82981C8.52795 4.66667 7.90752 4.66667 6.66667 4.66667C5.75147 4.66667 5.17377 4.66667 4.73212 4.73212M11.2679 11.2679C11.2446 11.425 11.213 11.5649 11.1702 11.6967C10.8405 12.7115 10.0449 13.5071 9.03006 13.8369C8.52795 14 7.90752 14 6.66667 14C5.42581 14 4.80539 14 4.30328 13.8369C3.28848 13.5071 2.49287 12.7115 2.16314 11.6967C2 11.1946 2 10.5742 2 9.33333C2 8.09248 2 7.47205 2.16314 6.96994C2.49287 5.95515 3.28848 5.15954 4.30328 4.82981C4.43505 4.787 4.57498 4.75542 4.73212 4.73212'
      />
    </svg>
  );
};
