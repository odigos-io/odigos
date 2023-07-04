"use client";
import styled, { ThemeProvider } from "styled-components";
import theme from "styles/palette";

// export const metadata = {
//   title: "odigos UI",
// };

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <ThemeProvider theme={theme}>
        <body style={{ margin: 0 }}>{children}</body>
      </ThemeProvider>
    </html>
  );
}
