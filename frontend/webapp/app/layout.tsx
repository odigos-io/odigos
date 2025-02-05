'use client';
import React, { PropsWithChildren } from 'react';
import { METADATA } from '@/utils';
import { ApolloWrapper } from '@/lib';
import { ThemeProvider } from '@/styles';
import { useDarkModeStore } from '@/store';

function RootLayout({ children }: PropsWithChildren) {
  const { darkMode } = useDarkModeStore();

  return (
    <html lang='en' suppressHydrationWarning={true}>
      <head>
        <meta name='description' content={METADATA.title} />
        <link rel='icon' type='image/svg+xml' href={`/${METADATA.icon}`} />
        <link rel='manifest' href='/manifest.json' />
        <title>{METADATA.title}</title>
      </head>

      {/*
        The "window" check is to prevent hydration errors,
        previously we had "suppressHydrationWarning={true}" on the "body" tag, but it was causing issues and had to be handled better.
      */}

      {typeof window === 'undefined' ? null : (
        <ApolloWrapper>
          <ThemeProvider darkMode={darkMode}>
            <body
              style={{
                width: '100vw',
                height: '100vh',
                margin: 0,
                backgroundColor: darkMode ? '#111111' : '#EEEEEE',
              }}
            >
              {children}
            </body>
          </ThemeProvider>
        </ApolloWrapper>
      )}
    </html>
  );
}

export default RootLayout;
