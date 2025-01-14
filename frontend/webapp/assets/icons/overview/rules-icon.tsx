import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const RulesIcon: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.info;

  return (
    <svg width={size} height={size} viewBox='0 0 16 16' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path
        stroke={fill}
        strokeWidth='1.2'
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M4.5 1.94059V1.94836M1.94111 4.49806V4.50583M1 7.99611V8.00389M1.94111 11.4942V11.5019M4.5 14.0516V14.0594M8 14.9922V15M11.5 14.0516V14.0594M14.0589 11.4942V11.5019M15 7.99611V8.00389M14.0589 4.49806V4.50583M11.5 1.94059V1.94836M8 1V1.00777'
      />
      <path
        stroke={fill}
        strokeLinecap='round'
        strokeLinejoin='round'
        d='M8 8H8.01125M8 3.5L9.326 4.79876L11.182 4.81802L11.2012 6.674L12.5 8L11.2012 9.326L11.182 11.182L9.326 11.2012L8 12.5L6.674 11.2012L4.81802 11.182L4.79876 9.326L3.5 8L4.79876 6.674L4.81802 4.81802L6.674 4.79876L8 3.5Z'
      />
    </svg>
  );
};
