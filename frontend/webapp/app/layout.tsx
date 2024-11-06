'use client';
import './globals.css';
import React from 'react';
import { useSSE } from '@/hooks';
import theme from '@/styles/theme';
import { ApolloWrapper } from '@/lib';
import { ThemeProvider } from 'styled-components';
import { QueryClient, QueryClientProvider } from 'react-query';

const LAYOUT_STYLE: React.CSSProperties = {
  margin: 0,
  position: 'fixed',
  scrollbarWidth: 'none',
  width: '100vw',
  height: '100vh',
};

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10000,
      refetchOnWindowFocus: false,
    },
  },
});

export default function RootLayout({ children }: { children: React.ReactNode }) {
  useSSE();

  return (
    <html lang='en'>
      <ApolloWrapper>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider theme={theme}>
            <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
              {children}
            </body>
          </ThemeProvider>
        </QueryClientProvider>
      </ApolloWrapper>
    </html>
  );
}
