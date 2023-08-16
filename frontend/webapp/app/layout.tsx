'use client';
import React from 'react';
import { ThemeProvider } from 'styled-components';
import theme from '@/styles/palette';
import { QueryClient, QueryClientProvider } from 'react-query';
import { ThemeProviderWrapper } from '@keyval-dev/design-system';

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
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 10000,
        refetchOnWindowFocus: false,
      },
    },
  });

  return (
    <html lang="en">
      <QueryClientProvider client={queryClient}>
        <ThemeProvider theme={theme}>
          <ThemeProviderWrapper>
            <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
              {children}
            </body>
          </ThemeProviderWrapper>
        </ThemeProvider>
      </QueryClientProvider>
    </html>
  );
}
