import React from 'react';
import styled from 'styled-components';

interface TextProps {
  children: React.ReactNode;
  color?: string;
  size?: number;
  weight?: number;
  align?: 'left' | 'center' | 'right';
  family?: 'primary' | 'secondary';
  opacity?: number;
}

const TextWrapper = styled.div<{
  color?: string;
  size: number;
  weight: number;
  align: 'left' | 'center' | 'right';
  family?: 'primary' | 'secondary';
  opacity: number;
}>`
  color: ${({ color, theme }) => color || theme.colors.text};
  font-size: ${({ size }) => size}px;
  font-weight: ${({ weight }) => weight};
  text-align: ${({ align }) => align};
  opacity: ${({ opacity }) => opacity};
  font-family: ${({ theme, family }) => {
    if (family === 'primary') {
      return theme.font_family.primary;
    }
    if (family === 'secondary') {
      return theme.font_family.secondary;
    }
    return theme.font_family.primary;
  }};
`;

const Text: React.FC<TextProps> = ({
  children,
  color,
  size = 16,
  weight = 300,
  align = 'left',
  family = 'primary',
  opacity = 1,
}) => {
  return (
    <TextWrapper
      family={family}
      color={color}
      size={size}
      weight={weight}
      align={align}
      opacity={opacity}
    >
      {children}
    </TextWrapper>
  );
};

export { Text };
