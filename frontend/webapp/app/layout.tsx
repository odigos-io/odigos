'use client';
import React, { type PropsWithChildren } from 'react';
import { ApolloWrapper } from '@/lib';
import { ThemeProvider } from '@/styles';
import { useDarkModeStore } from '@/store';

const METADATA = {
  title: 'Odigos',
  icon: 'favicon.svg',
};

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = useDarkModeStore();

  const bodyStyle = {
    width: '100vw',
    height: '100vh',
    margin: 0,
    backgroundColor: darkMode ? '#111111' : '#EEEEEE',
  };

  return (
    <html lang='en'>
      <head>
        <meta name='description' content={METADATA.title} />
        <link rel='icon' type='image/svg+xml' href={`/${METADATA.icon}`} />
        <link rel='manifest' href='/manifest.json' />
        <title>{METADATA.title}</title>
      </head>

      <ApolloWrapper>
        <ThemeProvider darkMode={darkMode}>
          <body suppressHydrationWarning={true} style={bodyStyle}>
            {children}
          </body>
        </ThemeProvider>
      </ApolloWrapper>
    </html>
  );
}

export default RootLayout;
