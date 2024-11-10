'use client';
import './globals.css';
import React from 'react';
import { useSSE } from '@/hooks';
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
