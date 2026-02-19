/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  webpack: (config) => {
    // Alias mapbox-gl → maplibre-gl so @mapbox/mapbox-gl-draw resolves correctly
    config.resolve.alias = {
      ...config.resolve.alias,
      "mapbox-gl": "maplibre-gl",
    };
    return config;
  },
};

export default nextConfig;
