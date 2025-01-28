import React, { type PropsWithChildren } from 'react';
import { useDarkModeStore } from '@/store';
import { Theme } from '@odigos/ui-components';
import StyledComponentsRegistry from './registry';

export const ThemeProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const { darkMode } = useDarkModeStore();

  return (
    <Theme.Provider darkMode={darkMode}>
      <StyledComponentsRegistry>{children}</StyledComponentsRegistry>
    </Theme.Provider>
  );
};
