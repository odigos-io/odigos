'use client';
import './globals.css';
import React from 'react';
import { useSSE } from '@/hooks';
import theme from '@/styles/theme';
import { ApolloWrapper } from '@/lib';
import { ThemeProvider } from 'styled-components';

const LAYOUT_STYLE: React.CSSProperties = {
  margin: 0,
  position: 'fixed',
  scrollbarWidth: 'none',
  width: '100vw',
  height: '100vh',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  useSSE();

  return (
    <html lang="en">
      <ApolloWrapper>
        <ThemeProvider theme={theme}>
          <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
            {children}
          </body>
        </ThemeProvider>
      </ApolloWrapper>
    </html>
  );
}
