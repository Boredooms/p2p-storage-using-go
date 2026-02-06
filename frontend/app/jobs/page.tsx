'use client';

import { useStore } from '@/lib/store';
import JobCard from '@/components/JobCard';

export default function JobsPage() {
    const { jobs } = useStore();

    return (
        <main className="min-h-screen bg-gradient-to-br from-gray-900 via-purple-900 to-black text-white">
            <div className="container mx-auto px-4 py-8">
                {/* Header */}
                <div className="mb-8">
                    <h1 className="text-4xl font-bold mb-2 bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-pink-600">
                        Job Dashboard
                    </h1>
                    <p className="text-gray-400">Track your compute jobs in real-time</p>
                </div>

                {/* Stats */}
                <div className="grid md:grid-cols-4 gap-4 mb-8">
                    {[
                        { label: 'Total Jobs', value: jobs.length, icon: '📊' },
                        { label: 'Running', value: jobs.filter(j => j.status === 'running').length, icon: '⚡' },
                        { label: 'Complete', value: jobs.filter(j => j.status === 'complete').length, icon: '✅' },
                        { label: 'Failed', value: jobs.filter(j => j.status === 'failed').length, icon: '❌' },
                    ].map((stat) => (
                        <div key={stat.label} className="bg-white/5 backdrop-blur-lg border border-white/10 rounded-lg p-4">
                            <div className="flex items-center gap-2 mb-2">
                                <span className="text-2xl">{stat.icon}</span>
                                <p className="text-sm text-gray-400">{stat.label}</p>
                            </div>
                            <p className="text-3xl font-bold">{stat.value}</p>
                        </div>
                    ))}
                </div>

                {/* Jobs List */}
                <div className="space-y-4">
                    {jobs.length === 0 ? (
                        <div className="text-center py-16">
                            <p className="text-6xl mb-4">📭</p>
                            <p className="text-xl text-gray-400 mb-2">No jobs yet</p>
                            <p className="text-sm text-gray-500 mb-6">Submit your first compute job to get started</p>
                            <a
                                href="/editor"
                                className="inline-block px-6 py-3 bg-purple-600 hover:bg-purple-500 rounded-lg font-medium transition-colors"
                            >
                                Go to Editor →
                            </a>
                        </div>
                    ) : (
                        jobs.map((job) => (
                            <JobCard key={job.id} {...job} />
                        ))
                    )}
                </div>
            </div>
        </main>
    );
}
