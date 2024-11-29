import React from 'react';
import styled from 'styled-components';

interface Props {
  orientation?: 'horizontal' | 'vertical';
  thickness?: number;
  length?: string;
  color?: string;
  margin?: string;
}

const StyledDivider = styled.div<{
  $orientation?: Props['orientation'];
  $thickness?: Props['thickness'];
  $length?: Props['length'];
  $color?: Props['color'];
  $margin?: Props['margin'];
}>`
  width: ${({ $orientation, $thickness, $length }) => ($orientation === 'vertical' ? `${$thickness}px` : $length || '100%')};
  height: ${({ $orientation, $thickness, $length }) => ($orientation === 'horizontal' ? `${$thickness}px` : $length || '100%')};
  margin: ${({ $orientation, $margin }) => $margin || ($orientation === 'horizontal' ? '8px 0' : '0 8px')};
  background-color: ${({ $color, theme }) => $color || theme.colors.border};
`;

const Divider: React.FC<Props> = ({ orientation = 'horizontal', thickness = 1, length, color, margin }) => {
  return <StyledDivider $orientation={orientation} $thickness={thickness} $length={length} $color={color} $margin={margin} />;
};

export { Divider };
