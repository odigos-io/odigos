/** @type {import('next').NextConfig} */

module.exports = {
  reactStrictMode: false,
  images: {
    unoptimized: true,
  },
  output: "export",
  webpack(config) {
    config.module.rules.push({
      test: /\.svg$/,
      use: {
        loader: "@svgr/webpack",
        options: {
          svgoConfig: {
            plugins: [
              {
                name: "removeViewBox",
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
