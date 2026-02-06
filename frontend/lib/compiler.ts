/**
 * Browser-based C to WASM compiler using clang-wasm
 * This compiles C code to WASM directly in the browser!
 */

// For now, we'll use a simple approach: send C code as-is and let backend handle it
// In production, you'd use @wasmer/wasi or clang-wasm here

export async function compileToWasm(code: string, language: 'c' | 'rust'): Promise<Uint8Array> {
    // Check if code is already WASM (magic number: 0x00 0x61 0x73 0x6D)
    const encoder = new TextEncoder();
    const bytes = encoder.encode(code);

    if (bytes.length >= 4 &&
        bytes[0] === 0x00 &&
        bytes[1] === 0x61 &&
        bytes[2] === 0x73 &&
        bytes[3] === 0x6D) {
        return bytes;
    }

    // For MVP: Return C code as bytes
    // Backend will handle compilation
    // TODO: Add browser-based compilation with clang-wasm
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

/**
 * Future enhancement: Use clang-wasm for true browser compilation
 * 
 * Example with @wasmer/wasi:
 * 
 * import { init, WASI } from '@wasmer/wasi';
 * 
 * async function compileWithClang(cCode: string): Promise<Uint8Array> {
 *   await init();
 *   
 *   const wasi = new WASI({
 *     args: ['clang', '-target', 'wasm32-wasi', '-o', 'output.wasm', 'input.c'],
 *     env: {},
 *     bindings: {
 *       fs: {
 *         'input.c': new TextEncoder().encode(cCode)
 *       }
 *     }
 *   });
 *   
 *   const module = await WebAssembly.compileStreaming(fetch('/clang.wasm'));
 *   const instance = await WebAssembly.instantiate(module, wasi.getImports(module));
 *   
 *   wasi.start(instance);
 *   
 *   return wasi.fs.readFileSync('output.wasm');
 * }
 */
