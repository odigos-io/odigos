import type { NextConfig } from 'next';

// Bundle analyzer configuration
const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});

const nextConfig: NextConfig = {
  output: 'export',
  reactStrictMode: false,
  images: {
    unoptimized: true,
  },
  compiler: {
    styledComponents: true,
    // Remove console.logs in production
    removeConsole: process.env.NODE_ENV === 'production',
  },
  // Enable compression
  compress: true,
  // Enable source maps only in development
  productionBrowserSourceMaps: false,
  // Enable experimental optimizations
  experimental: {
    // Enable tree shaking for better bundle optimization
    optimizePackageImports: ['@odigos/ui-kit', '@apollo/client', '@apollo/experimental-nextjs-app-support', 'graphql', 'react', 'react-dom', 'react-error-boundary', 'styled-components', 'zustand'],
  },
  // Turbopack configuration (empty config silences the warning)
  turbopack: {
    resolveAlias: {
      'styled-components': './node_modules/styled-components',
      zustand: './node_modules/zustand',
    },
  },
};

export default withBundleAnalyzer(nextConfig);
