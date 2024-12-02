import React from 'react';
import styled from 'styled-components';
import { hexPercentValues } from '@/styles';
import type { NotificationType } from '@/types';

interface Props {
  orientation?: 'horizontal' | 'vertical';
  type?: NotificationType;
  thickness?: number;
  length?: string;
  margin?: string;
}

const StyledDivider = styled.div<{
  $orientation?: Props['orientation'];
  $type?: Props['type'];
  $thickness?: Props['thickness'];
  $length?: Props['length'];
  $margin?: Props['margin'];
}>`
  width: ${({ $orientation, $thickness, $length }) => ($orientation === 'vertical' ? `${$thickness}px` : $length || '100%')};
  height: ${({ $orientation, $thickness, $length }) => ($orientation === 'horizontal' ? `${$thickness}px` : $length || '100%')};
  margin: ${({ $orientation, $margin }) => $margin || ($orientation === 'horizontal' ? '8px 0' : '0 8px')};
  background-color: ${({ $type, theme }) => (!!$type ? theme.text[$type] : theme.colors.border) + hexPercentValues['050']};
`;

export const Divider: React.FC<Props> = ({ orientation = 'horizontal', type, thickness = 1, length, margin }) => {
  return <StyledDivider $orientation={orientation} $type={type} $thickness={thickness} $length={length} $margin={margin} />;
};
