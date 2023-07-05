/** @type {import('next').NextConfig} */
// const nextConfig = {
//   reactStrictMode: true,
//   images: {
//     unoptimized: true,
//   },
//   async rewrites() {
//     // When running Next.js via Node.js (e.g. `dev` mode), proxy API requests
//     // to the Go server.
//     return [
//       {
//         source: '/api/:path*',
//         destination: 'http://localhost:8080/api/:path*',
//       },
//     ];
//   },
// }

// module.exports = nextConfig

module.exports = {
  reactStrictMode: true,
  output: "standalone",
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
