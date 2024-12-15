'use client';
import './globals.css';
import React from 'react';
import { useSSE } from '@/hooks';
import { METADATA } from '@/utils';
import { ApolloWrapper } from '@/lib';
import { ThemeProviderWrapper } from '@/styles';

const LAYOUT_STYLE: React.CSSProperties = {
  position: 'fixed',
  width: '100vw',
  height: '100vh',
  margin: 0,
  backgroundColor: '#111111',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  useSSE();

  return (
    <html lang='en'>
      <head>
        <meta name='description' content={METADATA.title} />
        <link rel='icon' type='image/svg+xml' href={`/${METADATA.icons}`} />
        <link rel='icon' type='image/x-icon' href='/favicon.ico' />
        <link rel='icon' type='image/png' sizes='16x16' href='/favicon-16x16.png' />
        <link rel='icon' type='image/png' sizes='32x32' href='/favicon-32x32.png' />
        <link rel='apple-touch-icon' sizes='180x180' href='/apple-touch-icon.png' />
        <link rel='manifest' href='/manifest.json' />
        <title>{METADATA.title}</title>
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
