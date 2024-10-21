'use client';
import './globals.css';
import React from 'react';
import { useSSE } from '@/hooks';
import theme from '@/styles/theme';
import { ApolloWrapper } from '@/lib';
import { ThemeProvider } from 'styled-components';
import { NotificationManager } from '@/components';
import ReduxProvider from '@/store/redux-provider';
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

  useSSE();

  return (
    <html lang="en">
      <ReduxProvider>
        <ApolloWrapper>
          <QueryClientProvider client={queryClient}>
            <ThemeProvider theme={theme}>
              {/* <ThemeProviderWrapper> */}
              <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
                {children}
                {/* <NotificationManager /> */}
              </body>
              {/* </ThemeProviderWrapper> */}
            </ThemeProvider>
          </QueryClientProvider>
        </ApolloWrapper>
      </ReduxProvider>
    </html>
  );
}
