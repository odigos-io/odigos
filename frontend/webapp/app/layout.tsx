"use client";
import React from "react";
import { ThemeProvider } from "styled-components";
import theme from "@/styles/palette";
import { QueryClient, QueryClientProvider } from "react-query";
import Head from "next/head";

const LAYOUT_STYLE: React.CSSProperties = {
  margin: 0,
  position: "fixed",
  scrollbarWidth: "none",
  width: "100vw",
  height: "100vh",
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
          <Head>
            <title>Odigos</title>
          </Head>
          <body suppressHydrationWarning={true} style={LAYOUT_STYLE}>
            {children}
          </body>
        </ThemeProvider>
      </QueryClientProvider>
    </html>
  );
}
