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
    optimizePackageImports: ['@apollo/client', 'graphql', 'zustand'],
  },
  // Webpack configuration for bundle optimization
  webpack: (config, { dev, isServer }) => {
    if (!dev && !isServer) {
      // Enable production optimizations
      config.optimization = {
        ...config.optimization,
        // Enable tree shaking
        usedExports: true,
        // Enable side effects optimization
        sideEffects: true,
        // Split chunks more aggressively
        splitChunks: {
          ...config.optimization.splitChunks,
          chunks: 'all',
          minSize: 20000,
          maxSize: 244000,
          cacheGroups: {
            vendor: {
              test: /[\\/]node_modules[\\/]/,
              name: 'vendors',
              chunks: 'all',
              priority: 10,
              enforce: true,
            },
            common: {
              name: 'common',
              minChunks: 2,
              chunks: 'all',
              priority: 5,
              reuseExistingChunk: true,
            },
            // Separate styled-components
            styledComponents: {
              test: /[\\/]node_modules[\\/]styled-components[\\/]/,
              name: 'styled-components',
              chunks: 'all',
              priority: 20,
              enforce: true,
            },
            // Separate Apollo Client
            apollo: {
              test: /[\\/]node_modules[\\/]@apollo[\\/]/,
              name: 'apollo',
              chunks: 'all',
              priority: 15,
              enforce: true,
            },
          },
        },
        // Better minification
        minimize: true,
        minimizer: config.optimization.minimizer.map((minimizer: any) => {
          if (minimizer.constructor.name === 'TerserPlugin') {
            return new (require('terser-webpack-plugin'))({
              terserOptions: {
                compress: {
                  drop_console: true,
                  drop_debugger: true,
                  pure_funcs: ['console.log', 'console.info', 'console.debug', 'console.warn'],
                },
                mangle: true,
                format: {
                  comments: false,
                },
              },
              extractComments: false,
            });
          }
          return minimizer;
        }),
      };

      // Add module concatenation
      config.optimization.concatenateModules = true;

      // Enable scope hoisting
      config.optimization.mergeDuplicateChunks = true;
    }
    return config;
  },
  // Enable compression
  compress: true,
  // Enable source maps only in development
  productionBrowserSourceMaps: false,
};

export default withBundleAnalyzer(nextConfig);
