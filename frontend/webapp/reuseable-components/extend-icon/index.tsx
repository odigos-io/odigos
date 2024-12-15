import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';

interface Props {
  extend: boolean;
  size?: number;
  align?: 'left' | 'right' | 'center';
}

const Icon = styled(Image)<{ $align?: Props['align'] }>`
  margin: ${({ $align }) => ($align === 'right' ? 'auto 0 auto auto' : $align === 'left' ? 'auto auto auto 0' : 'auto')};
  &.open {
    transform: rotate(180deg);
  }
  &.close {
    transform: rotate(0deg);
  }
  transition: transform 0.3s;
`;

export const ExtendIcon: React.FC<Props> = ({ extend, size = 14, align = 'center' }) => {
  return <Icon src='/icons/common/extend-arrow.svg' alt='extend' width={size} height={size} $align={align} className={extend ? 'open' : 'close'} />;
};
