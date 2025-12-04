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
  // Enable experimental optimizations
  experimental: {
    // Enable tree shaking for better bundle optimization
    optimizePackageImports: ['@odigos/ui-kit', '@apollo/client', 'graphql', 'zustand', 'styled-components', 'react', 'react-dom'],
    // Enable Turbopack file system caching for faster builds (dev only)
    turbopackFileSystemCacheForDev: true,
  },
  // Turbopack configuration (for dev mode - empty config silences the warning)
  turbopack: {},
  // Enable compression
  compress: true,
  // Enable source maps only in development
  productionBrowserSourceMaps: false,
};

export default withBundleAnalyzer(nextConfig);
