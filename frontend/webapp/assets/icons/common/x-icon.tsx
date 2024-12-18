import React from 'react';
import { SVG } from '@/assets';
import theme from '@/styles/theme';

export const XIcon: SVG = ({ size = 16, fill = theme.text.secondary, rotate = 0, onClick }) => {
  return (
    <svg width={size} height={size * (14 / 18)} viewBox='0 0 18 14' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path fill={fill} d='M9.487 5.832v2.433h1.495V5.832H9.487ZM17.847 17.185l2.028-1.791L9.487 7.047l10.388-8.414L17.928-3.158 6.162 6.304v1.453l11.685 9.428Z' />
      <path fill={fill} d='M8.513 5.832v2.433H7.018V5.832h1.495ZM0.153 17.185l-2.028-1.791L8.513 7.047l-10.388-8.414L0.072-3.158l11.766 9.462v1.453L0.153 17.185Z' />
    </svg>
  );
};
