// API Configuration - Auto-detects Codespaces or Production
export const API_BASE_URL =
    process.env.NEXT_PUBLIC_API_URL ||  // Production (from .env)
    'https://p2p-compute-backend.onrender.com';  // Production Render URL

console.log('[API] Using endpoint:', API_BASE_URL);
