import CryptoJS from 'crypto-js';

// Generate a random encryption key
export function generateKey(): string {
    return CryptoJS.lib.WordArray.random(32).toString();
}

// Encrypt WASM code
export function encryptWasm(wasm: Uint8Array, key: string): string {
    const wordArray = CryptoJS.lib.WordArray.create(wasm as any);
    const encrypted = CryptoJS.AES.encrypt(wordArray.toString(CryptoJS.enc.Base64), key);
    return encrypted.toString();
}

// Decrypt WASM code
export function decryptWasm(encryptedData: string, key: string): Uint8Array {
    const decrypted = CryptoJS.AES.decrypt(encryptedData, key);
    const base64 = decrypted.toString(CryptoJS.enc.Utf8);
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

// Encrypt input data
export function encryptInput(input: string, key: string): string {
    return CryptoJS.AES.encrypt(input, key).toString();
}

// Decrypt result
export function decryptResult(encryptedResult: string, key: string): string {
    const decrypted = CryptoJS.AES.decrypt(encryptedResult, key);
    return decrypted.toString(CryptoJS.enc.Utf8);
}

// Hash for verification
export function hashData(data: string): string {
    return CryptoJS.SHA256(data).toString();
}
