import { create } from 'zustand';

interface Job {
    id: string;
    status: 'pending' | 'running' | 'complete' | 'failed';
    result?: string;
    error?: string;
    timestamp: number;
}

interface AppState {
    jobs: Job[];
    walletAddress: string | null;
    balance: number;
    encryptionKey: string | null;

    // Actions
    addJob: (job: Job) => void;
    updateJob: (id: string, updates: Partial<Job>) => void;
    setWallet: (address: string, balance: number) => void;
    setEncryptionKey: (key: string) => void;
}

export const useStore = create<AppState>((set) => ({
    jobs: [],
    walletAddress: null,
    balance: 0,
    encryptionKey: null,

    addJob: (job) => set((state) => ({ jobs: [job, ...state.jobs] })),

    updateJob: (id, updates) => set((state) => ({
        jobs: state.jobs.map(job =>
            job.id === id ? { ...job, ...updates } : job
        )
    })),

    setWallet: (address, balance) => set({ walletAddress: address, balance }),

    setEncryptionKey: (key) => set({ encryptionKey: key }),
}));
