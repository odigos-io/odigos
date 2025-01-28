import React, { type PropsWithChildren } from 'react';
import { useDarkModeStore } from '@/store';
import { theme } from '@odigos/ui-components';
import StyledComponentsRegistry from './registry';

export const ThemeProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const { darkMode } = useDarkModeStore();

  return (
    <theme.Provider theme={theme.getTheme(darkMode) as theme.ITheme}>
      <StyledComponentsRegistry>{children}</StyledComponentsRegistry>
    </theme.Provider>
  );
};
