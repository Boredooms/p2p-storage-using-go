'use client';

import { useState } from 'react';
import CodeEditor from '@/components/CodeEditor';
import WalletCard from '@/components/WalletCard';
import { useStore } from '@/lib/store';
import { encryptWasm, encryptInput, generateKey } from '@/lib/encryption';
import { submitJob } from '@/lib/api';
import { compileToWasm } from '@/lib/compiler';

const EXAMPLE_C_CODE = `#include <stdio.h>

int factorial(int n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);
}

int main() {
    int input = 5;
    int result = factorial(input);
    printf("Factorial of %d = %d\\n", input, result);
    return result;
}`;

export default function EditorPage() {
    const [code, setCode] = useState(EXAMPLE_C_CODE);
    const [language, setLanguage] = useState('c');
    const [input, setInput] = useState('5');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [useEncryption, setUseEncryption] = useState(true);

    const { addJob, setEncryptionKey } = useStore();

    const handleSubmit = async () => {
        setIsSubmitting(true);
        try {
            // Compile code to WASM (sends C code to backend for compilation)
            const wasmBytes = await compileToWasm(code, language as 'c' | 'rust');

            let finalWasm = wasmBytes;
            let finalInput = input;
            let encKey = '';

            if (useEncryption) {
                // Generate encryption key
                encKey = generateKey();
                setEncryptionKey(encKey);

                // Encrypt WASM and input
                const encryptedWasm = encryptWasm(wasmBytes, encKey);
                const encryptedInput = encryptInput(input, encKey);

                finalWasm = new TextEncoder().encode(encryptedWasm);
                finalInput = encryptedInput;
            }

            // Create job
            const jobId = `job_${Date.now()}`;
            addJob({
                id: jobId,
                status: 'pending',
                timestamp: Date.now(),
            });

            // Submit to backend (placeholder TX ID)
            const result = await submitJob(finalWasm, finalInput, 'TX_DEMO_123');

            // Update job status
            addJob({
                id: jobId,
                status: 'complete',
                result: JSON.stringify(result),
                timestamp: Date.now(),
            });

            alert('Job submitted successfully!');
        } catch (error) {
            console.error('Job submission failed:', error);
            alert('Job submission failed: ' + (error as Error).message);
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <main className="min-h-screen bg-gradient-to-br from-gray-900 via-purple-900 to-black text-white">
            <div className="container mx-auto px-4 py-8">
                {/* Header */}
                <div className="mb-8">
                    <h1 className="text-4xl font-bold mb-2 bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-pink-600">
                        Code Editor
                    </h1>
                    <p className="text-gray-400">Write code, compile to WASM, execute on decentralized network</p>
                </div>

                <div className="grid lg:grid-cols-3 gap-6">
                    {/* Left: Editor */}
                    <div className="lg:col-span-2 space-y-4">
                        {/* Language Selector */}
                        <div className="flex gap-2">
                            {['c', 'rust', 'javascript'].map((lang) => (
                                <button
                                    key={lang}
                                    onClick={() => setLanguage(lang)}
                                    className={`px-4 py-2 rounded-lg font-medium transition-all ${language === lang
                                        ? 'bg-purple-600 text-white'
                                        : 'bg-white/10 text-gray-400 hover:bg-white/20'
                                        }`}
                                >
                                    {lang.toUpperCase()}
                                </button>
                            ))}
                        </div>

                        {/* Code Editor */}
                        <CodeEditor
                            language={language}
                            defaultValue={EXAMPLE_C_CODE}
                            onChange={(value) => setCode(value || '')}
                        />

                        {/* Input */}
                        <div>
                            <label className="block text-sm text-gray-400 mb-2">Input Data:</label>
                            <input
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                className="w-full px-4 py-2 bg-white/10 border border-white/20 rounded-lg focus:border-purple-500 focus:outline-none"
                                placeholder="Enter input for your program"
                            />
                        </div>

                        {/* Encryption Toggle */}
                        <div className="flex items-center gap-3 p-4 bg-white/5 border border-white/10 rounded-lg">
                            <input
                                type="checkbox"
                                checked={useEncryption}
                                onChange={(e) => setUseEncryption(e.target.checked)}
                                className="w-5 h-5"
                            />
                            <div>
                                <p className="font-medium">🔐 Enable Encryption</p>
                                <p className="text-xs text-gray-400">
                                    Encrypt your code before sending to compute nodes (AES-256)
                                </p>
                            </div>
                        </div>

                        {/* Submit Button */}
                        <button
                            onClick={handleSubmit}
                            disabled={isSubmitting}
                            className="w-full py-4 bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-500 hover:to-pink-500 rounded-lg font-bold text-lg disabled:opacity-50 disabled:cursor-not-allowed transition-all"
                        >
                            {isSubmitting ? '⏳ Submitting...' : '🚀 Compile & Execute'}
                        </button>
                    </div>

                    {/* Right: Wallet & Info */}
                    <div className="space-y-4">
                        <WalletCard />

                        {/* Info Card */}
                        <div className="bg-white/5 backdrop-blur-lg border border-white/10 rounded-lg p-6">
                            <h3 className="font-bold mb-3">How it works:</h3>
                            <ol className="space-y-2 text-sm text-gray-300">
                                <li>1. Write your code in C/Rust</li>
                                <li>2. Code is compiled to WASM</li>
                                <li>3. Encrypted with AES-256</li>
                                <li>4. Sent to compute nodes</li>
                                <li>5. Results returned encrypted</li>
                            </ol>
                        </div>
                    </div>
                </div>
            </div>
        </main>
    );
}
