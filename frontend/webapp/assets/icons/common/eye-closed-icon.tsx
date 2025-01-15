import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const EyeClosedIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg width={size} height={size} viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M14.83 9.17999C14.2706 8.61995 13.5576 8.23846 12.7813 8.08386C12.0049 7.92926 11.2002 8.00851 10.4689 8.31152C9.73758 8.61453 9.11264 9.12769 8.67316 9.78607C8.23367 10.4444 7.99938 11.2184 8 12.01C7.99916 13.0663 8.41619 14.08 9.16004 14.83'
      />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M12 16.01C13.0609 16.01 14.0783 15.5886 14.8284 14.8384C15.5786 14.0883 16 13.0709 16 12.01' />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M17.61 6.39004L6.38 17.62C4.6208 15.9966 3.14099 14.0944 2 11.99C6.71 3.76002 12.44 1.89004 17.61 6.39004Z' />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M20.9994 3L17.6094 6.39' />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M6.38 17.62L3 21' />
      <path stroke={fill} strokeLinecap='round' strokeLinejoin='round' d='M19.5695 8.42999C20.4801 9.55186 21.2931 10.7496 21.9995 12.01C17.9995 19.01 13.2695 21.4 8.76953 19.23' />
    </svg>
  );
};
