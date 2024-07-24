import React from 'react';
import styled from 'styled-components';

interface TextProps {
  children: React.ReactNode;
  color?: string;
  size?: number;
  weight?: number;
  align?: 'left' | 'center' | 'right';
}

const TextWrapper = styled.span<{
  color?: string;
  size: number;
  weight: number;
  align: 'left' | 'center' | 'right';
}>`
  color: ${({ color, theme }) =>
    color || console.log({ theme }) || theme.colors.text};
  font-size: ${({ size }) => size}px;
  font-weight: ${({ weight }) => weight};
  text-align: ${({ align }) => align};
  font-family: ${({ theme }) => theme.font_family.primary};
`;

const Text: React.FC<TextProps> = ({
  children,
  color,
  size = 16,
  weight = 400,
  align = 'left',
}) => {
  return (
    <TextWrapper color={color} size={size} weight={weight} align={align}>
      {children}
    </TextWrapper>
  );
};

export { Text };
