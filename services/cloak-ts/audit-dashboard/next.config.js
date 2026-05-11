/** @type {import('next').NextConfig} */
const nextConfig = {
  env: {
    NEXT_PUBLIC_CLOAK_SERVICE_URL:
      process.env.NEXT_PUBLIC_CLOAK_SERVICE_URL ?? "http://localhost:4000",
  },
};

module.exports = nextConfig;
