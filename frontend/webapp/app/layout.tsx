'use client';
import React, { useEffect } from 'react';
import { ThemeProvider } from 'styled-components';
import theme from '@/styles/palette';
import { QueryClient, QueryClientProvider } from 'react-query';
import { ThemeProviderWrapper } from '@keyval-dev/design-system';
import ReduxProvider from '@/store/redux-provider';
import { useNotify } from '@/hooks';
import NotificationManager from '@/components/notification/notification-manager';

const LAYOUT_STYLE: React.CSSProperties = {
  margin: 0,
  position: 'fixed',
  scrollbarWidth: 'none',
  width: '100vw',
  height: '100vh',
  backgroundColor: theme.colors.dark,
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

  const notify = useNotify();
  useEffect(() => {
    const eventSource = new EventSource('http://localhost:8085/events');

    eventSource.onmessage = function (event) {
      const data = JSON.parse(event.data);
      notify(data.data, 'destination', 'success');
    };

    eventSource.onerror = function (event) {
      console.error('EventSource failed:', event);
    };

    // Clean up event source on component unmount
    return () => {
      eventSource.close();
    };
  }, []);

  return (
    <html lang="en">
      <ReduxProvider>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider theme={theme}>
            <ThemeProviderWrapper>
              <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
                {children}
                <NotificationManager />
              </body>
            </ThemeProviderWrapper>
          </ThemeProvider>
        </QueryClientProvider>
      </ReduxProvider>
    </html>
  );
}
