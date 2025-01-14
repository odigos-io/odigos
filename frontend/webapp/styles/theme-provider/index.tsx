import React, { type PropsWithChildren } from 'react';
import { getTheme } from '../theme';
import { useDarkModeStore } from '@/store';
import { ThemeProvider } from 'styled-components';
import StyledComponentsRegistry from './registry';

export const ThemeProviderWrapper: React.FC<PropsWithChildren> = ({ children }) => {
  const { darkMode } = useDarkModeStore();

  return (
    <ThemeProvider theme={getTheme(darkMode)}>
      <StyledComponentsRegistry>{children}</StyledComponentsRegistry>
    </ThemeProvider>
  );
};
