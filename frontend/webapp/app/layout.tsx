'use client';

import React, { type PropsWithChildren, useEffect } from 'react';
import dynamic from 'next/dynamic';
import { useDarkMode } from '@odigos/ui-kit/store';
import ApolloProvider from '@/lib/apollo-provider';
import useHiringConsoleMessage from '@/hooks/common/useHiringConsoleMessage';

const ThemeProvider = dynamic(() => import('@/lib/theme-provider'), { ssr: false });

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = useDarkMode();

  useHiringConsoleMessage();

  return (
    <html lang='en'>
      <head>
        <link rel='manifest' href='/manifest.json' />
        <link rel='icon' href='/favicon.svg' type='image/svg' />
        <meta name='description' content='Odigos' />
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
        <ThemeProvider>
          <ApolloProvider>{children}</ApolloProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}

export default RootLayout;
