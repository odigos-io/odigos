'use client';
import './globals.css';
import React from 'react';
import { METADATA } from '@/utils';
import { ApolloWrapper } from '@/lib';
import { useDarkModeStore } from '@/store';
import { ThemeProviderWrapper } from '@/styles';

export default function RootLayout({ children }: { children: React.ReactNode }) {
  const { darkMode } = useDarkModeStore();

  return (
    <html lang='en'>
      <head>
        <meta name='description' content={METADATA.title} />
        <link rel='icon' type='image/svg+xml' href={`/${METADATA.icon}`} />
        <link rel='manifest' href='/manifest.json' />
        <title>{METADATA.title}</title>
      </head>
      <ApolloWrapper>
        <ThemeProviderWrapper>
          <body
            suppressHydrationWarning={true}
            style={{
              width: '100vw',
              height: '100vh',
              margin: 0,
              backgroundColor: darkMode ? '#111111' : '#EEEEEE',
            }}
          >
            {children}
          </body>
        </ThemeProviderWrapper>
      </ApolloWrapper>
    </html>
  );
}
