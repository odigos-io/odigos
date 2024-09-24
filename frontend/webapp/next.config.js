/** @type {import('next').NextConfig} */

const PnpWebpackPlugin = require('pnp-webpack-plugin');

module.exports = {
  reactStrictMode: false,
  images: {
    unoptimized: true,
  },
  output: 'export',
  webpack(config) {
    // Add pnp-webpack-plugin to the resolver
    config.resolve.plugins = config.resolve.plugins || [];
    config.resolve.plugins.push(PnpWebpackPlugin);

    // Add pnp-webpack-plugin to the loader resolver
    config.resolveLoader = config.resolveLoader || {};
    config.resolveLoader.plugins = config.resolveLoader.plugins || [];
    config.resolveLoader.plugins.push(PnpWebpackPlugin.moduleLoader(module));

    // SVG handling with SVGR
    config.module.rules.push({
      test: /\.svg$/,
      use: {
        loader: '@svgr/webpack',
        options: {
          svgoConfig: {
            plugins: [
              {
                name: 'removeViewBox',
                active: false,
              },
            ],
          },
        },
      },
    });

    return config;
  },
};
