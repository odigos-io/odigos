'use client';

import React, { type PropsWithChildren } from 'react';
import dynamic from 'next/dynamic';
import { useDarkMode } from '@odigos/ui-kit/store';
import ApolloProvider from '@/lib/apollo-provider';

const ThemeProvider = dynamic(() => import('@/lib/theme-provider'), { ssr: false });

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = useDarkMode();

  return (
    <html lang='en'>
      <head>
        <link rel='manifest' href='/manifest.json' />
        <link rel='icon' type='image/x-icon' href='/favicon.svg' />
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
