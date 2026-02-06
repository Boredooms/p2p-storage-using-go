'use client';

interface JobCardProps {
    id: string;
    status: 'pending' | 'running' | 'complete' | 'failed';
    result?: string;
    error?: string;
    timestamp: number;
}

export default function JobCard({ id, status, result, error, timestamp }: JobCardProps) {
    const statusColors = {
        pending: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
        running: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
        complete: 'bg-green-500/20 text-green-400 border-green-500/30',
        failed: 'bg-red-500/20 text-red-400 border-red-500/30',
    };

    const statusIcons = {
        pending: '⏳',
        running: '⚡',
        complete: '✅',
        failed: '❌',
    };

    return (
        <div className="bg-white/5 backdrop-blur-lg border border-white/10 rounded-lg p-4 hover:border-purple-500/50 transition-all">
            <div className="flex justify-between items-start mb-3">
                <div className="flex items-center gap-2">
                    <span className="text-2xl">{statusIcons[status]}</span>
                    <div>
                        <h3 className="font-mono text-sm text-gray-400">Job #{id.slice(0, 8)}</h3>
                        <p className="text-xs text-gray-500">
                            {new Date(timestamp).toLocaleString()}
                        </p>
                    </div>
                </div>
                <span className={`px-3 py-1 rounded-full text-xs font-medium border ${statusColors[status]}`}>
                    {status.toUpperCase()}
                </span>
            </div>

            {result && (
                <div className="mt-3 p-3 bg-black/30 rounded border border-green-500/20">
                    <p className="text-xs text-gray-400 mb-1">Result:</p>
                    <pre className="text-sm text-green-400 font-mono overflow-x-auto">{result}</pre>
                </div>
            )}

            {error && (
                <div className="mt-3 p-3 bg-black/30 rounded border border-red-500/20">
                    <p className="text-xs text-gray-400 mb-1">Error:</p>
                    <pre className="text-sm text-red-400 font-mono overflow-x-auto">{error}</pre>
                </div>
            )}
        </div>
    );
}
