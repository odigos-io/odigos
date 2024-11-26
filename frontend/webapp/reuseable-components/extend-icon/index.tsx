import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';

interface Props {
  extend: boolean;
  size?: number;
  align?: 'left' | 'right';
}

const Icon = styled(Image)<{ $align?: Props['align'] }>`
  &.open {
    transform: rotate(180deg);
  }
  &.close {
    transform: rotate(0deg);
  }
  transition: transform 0.3s;
  margin-${({ $align }) => ($align === 'right' ? 'left' : 'right')}: auto;
`;

export const ExtendIcon: React.FC<Props> = ({ extend, size = 14, align = 'right' }) => {
  return <Icon src='/icons/common/extend-arrow.svg' alt='extend' width={size} height={size} $align={align} className={extend ? 'open' : 'close'} />;
};
