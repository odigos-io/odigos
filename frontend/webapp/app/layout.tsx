'use client';

import React, { type PropsWithChildren } from 'react';
import Theme from '@odigos/ui-theme';
import { ApolloWrapper } from '@/lib';
import { ThemeProvider } from '@/styles';
import { ErrorBoundary } from '@/components';

const METADATA = {
  title: 'Odigos',
  icon: 'favicon.svg',
};

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = Theme.useDarkMode();

  return (
    <html lang='en'>
      <head>
        <meta name='description' content={METADATA.title} />
        <link rel='icon' type='image/svg+xml' href={`/${METADATA.icon}`} />
        <link rel='manifest' href='/manifest.json' />
        <title>{METADATA.title}</title>
      </head>

      <body
        suppressHydrationWarning={true}
        style={{
          width: '100vw',
          height: '100vh',
          margin: 0,
          backgroundColor: darkMode ? '#111111' : '#EEEEEE',
        }}
      >
        <ErrorBoundary>
          <ApolloWrapper>
            <ThemeProvider>{children}</ThemeProvider>
          </ApolloWrapper>
        </ErrorBoundary>
      </body>
    </html>
  );
}

export default RootLayout;
