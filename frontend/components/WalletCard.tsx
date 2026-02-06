'use client';

import { useStore } from '@/lib/store';

export default function WalletCard() {
    const { walletAddress, balance } = useStore();

    return (
        <div className="bg-gradient-to-br from-purple-900/30 to-pink-900/30 backdrop-blur-lg border border-purple-500/30 rounded-lg p-6">
            <div className="flex items-center gap-3 mb-4">
                <div className="text-4xl">💰</div>
                <div>
                    <h3 className="text-lg font-bold">Wallet</h3>
                    <p className="text-xs text-gray-400">Your compute balance</p>
                </div>
            </div>

            {walletAddress ? (
                <>
                    <div className="mb-3">
                        <p className="text-xs text-gray-400 mb-1">Address:</p>
                        <p className="font-mono text-sm text-purple-300 break-all">
                            {walletAddress.slice(0, 20)}...{walletAddress.slice(-20)}
                        </p>
                    </div>
                    <div>
                        <p className="text-xs text-gray-400 mb-1">Balance:</p>
                        <p className="text-3xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-pink-400">
                            {balance} tokens
                        </p>
                    </div>
                </>
            ) : (
                <div className="text-center py-4">
                    <p className="text-gray-400 text-sm mb-3">No wallet connected</p>
                    <button className="px-4 py-2 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm font-medium transition-colors">
                        Generate Wallet
                    </button>
                </div>
            )}
        </div>
    );
}
