import React, { ReactNode } from 'react';
import StyledComponentsRegistry from './registry';
import { ThemeProvider } from 'styled-components';
import theme from '../theme';
interface ThemeProviderWrapperProps {
  children: ReactNode; // Add children prop with ReactNode type
}

export const ThemeProviderWrapper: React.FC<ThemeProviderWrapperProps> = ({ children }) => {
  return (
    <ThemeProvider theme={theme}>
      <StyledComponentsRegistry>{children}</StyledComponentsRegistry>
    </ThemeProvider>
  );
};
