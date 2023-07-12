"use client";
import React, { useState } from "react";
import { ThemeProvider } from "styled-components";
import theme from "@/styles/palette";
import { QueryClient, QueryClientProvider } from "react-query";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [queryClient] = useState(() => new QueryClient());

  return (
    <html lang="en">
      <QueryClientProvider client={queryClient}>
        <ThemeProvider theme={theme}>
          <body
            suppressHydrationWarning={true}
            style={{ margin: 0, position: "fixed" }}
          >
            {children}
          </body>
        </ThemeProvider>
      </QueryClientProvider>
    </html>
  );
}
