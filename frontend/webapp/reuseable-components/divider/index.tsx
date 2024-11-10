import React from 'react';
import styled from 'styled-components';

interface DividerProps {
  thickness?: number;
  color?: string;
  margin?: string;
  orientation?: 'horizontal' | 'vertical';
}

const StyledDivider = styled.div<DividerProps>`
  width: ${({ orientation, thickness }) => (orientation === 'vertical' ? `${thickness}px` : '100%')};
  height: ${({ orientation, thickness }) => (orientation === 'horizontal' ? `${thickness}px` : '100%')};
  background-color: ${({ color, theme }) => color || theme.colors.border};
  margin: ${({ orientation, margin }) => margin || (orientation === 'horizontal' ? '8px 0' : '0 8px')};
`;

const Divider: React.FC<DividerProps> = ({ thickness = 1, color, margin, orientation = 'horizontal' }) => {
  return <StyledDivider thickness={thickness} color={color} margin={margin} orientation={orientation} />;
};

export { Divider };
