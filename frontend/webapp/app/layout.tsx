'use client';

import React, { type PropsWithChildren } from 'react';
import { useDarkMode } from '@odigos/ui-kit/store';
import dynamic from 'next/dynamic';

const ApolloProvider = dynamic(() => import('@/lib/apollo-provider'), { ssr: false });
const ThemeProvider = dynamic(() => import('@/lib/theme-provider'), { ssr: false });

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = useDarkMode();

  return (
    <html lang='en'>
      <head>
        <link rel='manifest' href='/manifest.json' />
        <link rel='icon' href='/favicon.svg' type='image/svg' />
        <meta name='description' content='Odigos' />
        <title>Odigos UI</title>
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
