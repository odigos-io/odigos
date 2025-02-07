'use client';

import React, { type PropsWithChildren } from 'react';
import dynamic from 'next/dynamic';
import Theme from '@odigos/ui-theme';
import { ApolloWrapper } from '@/lib';
import { ErrorBoundary } from '@/components';

const ThemeProvider = dynamic(() => import('@/providers/theme-provider'), { ssr: false });

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = Theme.useDarkMode();

  return (
    <html lang='en'>
      <head>
        <meta name='description' content='Odigos' />
        <link rel='icon' type='image/svg+xml' href='/favicon.svg' />
        <link rel='manifest' href='/manifest.json' />
        <title>Odigos</title>
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
