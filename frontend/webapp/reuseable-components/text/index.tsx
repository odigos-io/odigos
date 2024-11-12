import React from 'react';
import styled from 'styled-components';

interface TextProps extends React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  children: React.ReactNode;
  color?: string;
  size?: number;
  weight?: number;
  align?: 'left' | 'center' | 'right';
  family?: 'primary' | 'secondary';
  opacity?: number;
  decoration?: string;
}

const TextWrapper = styled.div<{
  color?: string;
  size?: number;
  weight?: number;
  align?: 'left' | 'center' | 'right';
  family?: 'primary' | 'secondary';
  opacity?: number;
  decoration?: string;
}>`
  color: ${({ color, theme }) => color || theme.colors.text};
  font-size: ${({ size }) => (size !== undefined ? size : 16)}px;
  font-weight: ${({ weight }) => (weight !== undefined ? weight : 300)};
  text-align: ${({ align }) => align || 'left'};
  opacity: ${({ opacity }) => (opacity !== undefined ? opacity : 1)};
  text-decoration: ${({ decoration }) => decoration || 'none'};
<<<<<<< HEAD
=======
  text-transform: ${({ family }) => (family === 'secondary' ? 'uppercase' : 'none')};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  font-family: ${({ theme, family }) => {
    if (family === 'secondary') {
      return theme.font_family.secondary;
    }
    return theme.font_family.primary;
  }};
`;

const Text: React.FC<TextProps> = ({ children, ...props }) => {
  return <TextWrapper {...props}>{children}</TextWrapper>;
};

export { Text };
