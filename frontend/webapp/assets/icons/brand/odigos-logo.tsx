import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const OdigosLogo: SVG = ({ size = 16, fill: f, rotate = 0, onClick }) => {
  const theme = useTheme();
  const fill = f || theme.text.secondary;

  return (
    <svg xmlns='http://www.w3.org/2000/svg' width={size} height={size * (431 / 552)} viewBox='0 0 552 431' fill={fill} style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path d='M308.491 83.3091V0.0765381H472.573C493.785 0.0765381 511 16.4464 511 36.6589V374.508C511 394.72 493.785 411.09 472.573 411.09H308.491V328.396L454.666 263.762C478.721 253.079 493.708 230.791 493.708 205.583C493.708 180.375 478.721 158.011 454.666 147.405L308.414 83.3859L308.491 83.3091Z' />
      <path d='M202.51 327.781V411.014H38.4269C17.2152 411.014 0 394.644 0 374.431V36.5824C0 16.3698 17.2152 0 38.4269 0H202.51V82.6946L56.3338 147.329C32.2786 158.011 17.2921 180.299 17.2921 205.507C17.2921 230.715 32.2786 253.079 56.3338 263.685L202.586 327.704L202.51 327.781Z' />
      <path d='M255.462 290.507C302.363 290.507 340.385 252.485 340.385 205.584C340.385 158.682 302.363 120.66 255.462 120.66C208.56 120.66 170.538 158.682 170.538 205.584C170.538 252.485 208.56 290.507 255.462 290.507Z' />
    </svg>
  );
};
