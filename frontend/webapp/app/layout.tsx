'use client';

import React, { type PropsWithChildren } from 'react';
import dynamic from 'next/dynamic';
import Theme from '@odigos/ui-theme';
import ApolloProvider from '@/components/providers/apollo-provider';

const ThemeProvider = dynamic(() => import('@/components/providers/theme-provider'), { ssr: false });

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = Theme.useDarkMode();

  return (
    <html lang='en'>
      <head>
        <meta name='description' content='Odigos' />
        <link rel='icon' type='image/x-icon' href='/favicon.svg' />
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
        <ThemeProvider>
          <ApolloProvider>{children}</ApolloProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}

export default RootLayout;
