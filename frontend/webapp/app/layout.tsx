'use client';
import './globals.css';
import React from 'react';
import { useSSE } from '@/hooks';
import { METADATA } from '@/utils';
import { ApolloWrapper } from '@/lib';
import { ThemeProviderWrapper } from '@/styles';

const LAYOUT_STYLE: React.CSSProperties = {
  margin: 0,
  position: 'fixed',
  scrollbarWidth: 'none',
  width: '100vw',
  height: '100vh',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  useSSE();

  return (
    <html lang='en'>
      <head>
        <link rel='icon' href={`/${METADATA.icons}`} type='image/svg+xml' />
        <title>{METADATA.title}</title>
        <meta name='description' content={METADATA.title} />
      </head>
      <ApolloWrapper>
        <ThemeProviderWrapper>
          <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
            {children}
          </body>
        </ThemeProviderWrapper>
      </ApolloWrapper>
    </html>
  );
}
