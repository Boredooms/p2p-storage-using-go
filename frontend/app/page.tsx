export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-br from-gray-900 via-purple-900 to-black text-white">
      <div className="container mx-auto px-4 py-16">
        {/* Hero Section */}
        <div className="text-center mb-16">
          <h1 className="text-6xl font-bold mb-4 bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-pink-600">
            Decentralized P2P Compute
          </h1>
          <p className="text-xl text-gray-300 max-w-2xl mx-auto">
            Distributed storage, encrypted compute, and blockchain-powered payments. All in one platform.
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid md:grid-3 gap-8 max-w-6xl mx-auto mb-16">
          {/* Feature 1 */}
          <div className="bg-white/10 backdrop-blur-lg rounded-xl p-8 border border-white/20 hover:border-purple-500 transition-all">
            <div className="text-4xl mb-4">🗄️</div>
            <h3 className="text-2xl font-bold mb-2">Distributed Storage</h3>
            <p className="text-gray-300">
              Files split into 14 shards using Reed-Solomon encoding. Fault-tolerant and decentralized.
            </p>
          </div>

          {/* Feature 2 */}
          <div className="bg-white/10 backdrop-blur-lg rounded-xl p-8 border border-white/20 hover:border-purple-500 transition-all">
            <div className="text-4xl mb-4">🔐</div>
            <h3 className="text-2xl font-bold mb-2">Encrypted Compute</h3>
            <p className="text-gray-300">
              Execute WebAssembly code securely. Nodes can't see your data or algorithms.
            </p>
          </div>

          {/* Feature 3 */}
          <div className="bg-white/10 backdrop-blur-lg rounded-xl p-8 border border-white/20 hover:border-purple-500 transition-all">
            <div className="text-4xl mb-4">⛓️</div>
            <h3 className="text-2xl font-bold mb-2">Blockchain Payments</h3>
            <p className="text-gray-300">
              Custom PoW blockchain with mining economy. Pay for compute with mined tokens.
            </p>
          </div>
        </div>

        {/* CTA Buttons */}
        <div className="flex gap-4 justify-center">
          <a
            href="/editor"
            className="px-8 py-4 bg-gradient-to-r from-purple-600 to-pink-600 rounded-lg font-bold text-lg hover:scale-105 transition-transform"
          >
            Start Computing →
          </a>
          <a
            href="/jobs"
            className="px-8 py-4 bg-white/10 backdrop-blur-lg rounded-lg font-bold text-lg border border-white/20 hover:border-purple-500 transition-all"
          >
            View Jobs
          </a>
        </div>

        {/* Stats */}
        <div className="mt-16 grid grid-cols-3 gap-8 max-w-3xl mx-auto text-center">
          <div>
            <div className="text-4xl font-bold text-purple-400">10+4</div>
            <div className="text-gray-400">Erasure Coding</div>
          </div>
          <div>
            <div className="text-4xl font-bold text-purple-400">AES-256</div>
            <div className="text-gray-400">Encryption</div>
          </div>
          <div>
            <div className="text-4xl font-bold text-purple-400">WASM</div>
            <div className="text-gray-400">Sandboxed Execution</div>
          </div>
        </div>
      </div>
    </main>
  );
}
