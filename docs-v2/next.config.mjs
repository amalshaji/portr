import { createMDX } from "fumadocs-mdx/next";

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  output: 'export',
  reactStrictMode: true,
  experimental: {
    optimizePackageImports: ["fumadocs-ui", "geist"],
  },
  // Headers are not supported with static export
  // Remove headers() configuration for static export
};

export default withMDX(config);
