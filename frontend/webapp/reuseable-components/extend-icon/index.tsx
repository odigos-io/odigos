import React from 'react';
import styled from 'styled-components';
import { ExtendArrowIcon } from '@/assets';

interface Props {
  extend: boolean;
  size?: number;
  align?: 'left' | 'right' | 'center';
}

const Container = styled.div<{ $align?: Props['align'] }>`
  margin: ${({ $align }) => ($align === 'right' ? 'auto 0 auto auto' : $align === 'left' ? 'auto auto auto 0' : 'auto')};
`;

export const ExtendIcon: React.FC<Props> = ({ extend, size = 14, align = 'center' }) => {
  return (
    <Container $align={align}>
      <ExtendArrowIcon size={size} rotate={extend ? 180 : 0} />
    </Container>
  );
};
