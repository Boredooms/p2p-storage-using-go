/**
 * Browser-based C to WASM compiler
 * Uses WebAssembly.sh API for fast online compilation
 */

/**
 * Compile C code to WASM using WebAssembly.sh API
 * Falls back to backend compilation if API fails
 */
export async function compileToWasm(code: string, language: 'c' | 'rust'): Promise<Uint8Array> {
    const encoder = new TextEncoder();
    const bytes = encoder.encode(code);

    // Check if already WASM (magic number: 0x00 0x61 0x73 0x6D)
    if (bytes.length >= 4 &&
        bytes[0] === 0x00 &&
        bytes[1] === 0x61 &&
        bytes[2] === 0x73 &&
        bytes[3] === 0x6D) {
        console.log('[Compiler] Already WASM bytecode');
        return bytes;
    }

    // Try browser-based compilation for C code
    if (language === 'c') {
        try {
            console.log('[Compiler] Attempting browser-based C → WASM compilation...');

            // Use WasmFiddle API (free, no auth required)
            const response = await fetch('https://wasm.fastlylabs.com/compile', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    source: code,
                    language: 'c',
                    optimize: 2,
                }),
            });

            if (response.ok) {
                const result = await response.json();
                if (result.wasm) {
                    // Convert base64 WASM to Uint8Array
                    const wasmBase64 = result.wasm;
                    const wasmBinary = Uint8Array.from(atob(wasmBase64), c => c.charCodeAt(0));
                    console.log('[Compiler] ✅ Browser compilation successful!', wasmBinary.length, 'bytes');
                    return wasmBinary;
                }
            }

            console.log('[Compiler] Browser compilation unavailable, falling back to backend');
        } catch (error) {
            console.log('[Compiler] Browser compilation failed, falling back to backend:', error);
        }
    }

    // Fallback: send source code to backend for compilation
    console.log(`[Compiler] Sending ${language} source code to backend for compilation`);
    return bytes;
}

export function detectLanguage(code: string): 'c' | 'rust' | 'wasm' | 'unknown' {
    // Check for WASM magic number
    if (code.startsWith('\0asm')) {
        return 'wasm';
    }

    // Check for C patterns
    if (code.includes('#include') || code.includes('int main')) {
        return 'c';
    }

    // Check for Rust patterns
    if (code.includes('fn main') || code.includes('use std')) {
        return 'rust';
    }

    return 'unknown';
}
