import React, { type FC, type PropsWithChildren, useState } from 'react';
import Theme from '@odigos/ui-theme';
import { useServerInsertedHTML } from 'next/navigation';
import { ServerStyleSheet, StyleSheetManager } from 'styled-components';

const StyledComponentsRegistry: FC<PropsWithChildren> = ({ children }) => {
  // Only create stylesheet once with lazy initial state
  // x-ref: https://reactjs.org/docs/hooks-reference.html#lazy-initial-state
  const [styledComponentsStyleSheet] = useState(() => new ServerStyleSheet());

  useServerInsertedHTML(() => {
    const styles = styledComponentsStyleSheet.getStyleElement();
    styledComponentsStyleSheet.instance.clearTag();
    return <>{styles}</>;
  });

  if (typeof window !== 'undefined') return <>{children}</>;

  return <StyleSheetManager sheet={styledComponentsStyleSheet.instance}>{children}</StyleSheetManager>;
};

export const ThemeProvider: FC<PropsWithChildren> = ({ children }) => {
  return (
    <Theme.Provider>
      <StyledComponentsRegistry>{children}</StyledComponentsRegistry>
    </Theme.Provider>
  );
};
