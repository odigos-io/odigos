"use client";
import { ThemeProvider } from "styled-components";
import theme from "styles/palette";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <ThemeProvider theme={theme}>
        <body suppressHydrationWarning={true} style={{ margin: 0 }}>
          {children}
        </body>
      </ThemeProvider>
    </html>
  );
}
