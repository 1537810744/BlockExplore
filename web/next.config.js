/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: 'http://query-api:8080/api/v1/:path*',
      },
    ]
  },
}

module.exports = nextConfig
