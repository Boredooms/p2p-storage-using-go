// API Configuration - Auto-detects Codespaces or Production
export const API_BASE_URL =
    process.env.NEXT_PUBLIC_API_URL ||  // Production (from .env)
    'https://ubiquitous-telegram-pjpr5w6g5x6xc9p7w-8080.app.github.dev';  // Codespaces

console.log('[API] Using endpoint:', API_BASE_URL);
