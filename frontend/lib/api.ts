import { API_BASE_URL } from './config';

// Types
export interface Job {
    id: string;
    status: 'pending' | 'running' | 'complete' | 'failed';
    result?: string;
    error?: string;
}

export interface WalletInfo {
    address: string;
    balance: number;
}

// Job API
export async function submitJob(wasm: Uint8Array, input: string, txId: string): Promise<Job> {
    const response = await fetch(`${API_BASE_URL}/api/jobs/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            wasm: Array.from(wasm),
            input,
            paymentTx: txId
        })
    });

    if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
    }

    return response.json();
}

export async function getJobStatus(jobId: string): Promise<Job> {
    const response = await fetch(`${API_BASE_URL}/api/jobs/${jobId}`);

    if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
    }

    return response.json();
}

// Payment API
export async function makePayment(from: string, to: string, amount: number): Promise<{ txId: string }> {
    const response = await fetch(`${API_BASE_URL}/api/pay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ from, to, amount })
    });

    if (!response.ok) {
        throw new Error(`Payment failed: ${response.statusText}`);
    }

    return response.json();
}

// Wallet API
export async function getWalletInfo(address: string): Promise<WalletInfo> {
    const response = await fetch(`${API_BASE_URL}/api/wallet/${address}`);

    if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
    }

    return response.json();
}

// Storage API
export async function uploadFile(file: File): Promise<{ fileId: string; size: number }> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${API_BASE_URL}/api/storage/upload`, {
        method: 'POST',
        body: formData
    });

    if (!response.ok) {
        throw new Error(`Upload failed: ${response.statusText}`);
    }

    return response.json();
}

export async function downloadFile(fileId: string, size: number): Promise<Blob> {
    const response = await fetch(`${API_BASE_URL}/api/storage/download/${fileId}?size=${size}`);

    if (!response.ok) {
        throw new Error(`Download failed: ${response.statusText}`);
    }

    return response.blob();
}
