/** @type {import('next').NextConfig} */
const nextConfig = {
    webpack: (config, { isServer }) => {
        // Enable WASM support
        config.experiments = {
            ...config.experiments,
            asyncWebAssembly: true,
            layers: true,
        };

        // Handle .wasm files
        config.module.rules.push({
            test: /\.wasm$/,
            type: 'webassembly/async',
        });

        // Wasmer SDK requires these settings
        if (!isServer) {
            config.resolve.fallback = {
                ...config.resolve.fallback,
                fs: false,
                path: false,
                crypto: false,
            };
        }

        return config;
    },
};

export default nextConfig;
