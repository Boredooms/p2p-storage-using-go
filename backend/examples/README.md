# Multi-Language WASM Examples

This directory contains example code in different languages that can be compiled to WebAssembly and executed on the decentralized compute network.

## Quick Start

### 1. Install Emscripten (for C/C++)
```bash
git clone https://github.com/emscripten-core/emsdk.git
cd emsdk
./emsdk install latest
./emsdk activate latest
source ./emsdk_env.sh
```

### 2. Compile C Examples
```bash
cd examples

# Matrix multiplication
emcc matrix_multiply.c -o matrix_multiply.wasm -s STANDALONE_WASM -O3

# Prime checker
emcc prime_checker.c -o prime_checker.wasm -s STANDALONE_WASM -O3
```

### 3. Test Locally (Optional)
```bash
# Install wasmer
curl https://get.wasmer.io -sSfL | sh

# Run
wasmer run matrix_multiply.wasm
wasmer run prime_checker.wasm
```

### 4. Run on Decentralized Network
```bash
# Start node in Terminal 1
go run main.go

# Run job in Terminal 2
go run main.go run-job \
  --wasm examples/matrix_multiply.wasm \
  --input 0 \
  --tx <PAYMENT_TX_ID> \
  --peer <PEER_ADDR>
```

## Available Examples

- **matrix_multiply.c**: 3x3 matrix multiplication
- **prime_checker.c**: Prime number calculation up to 1000

## Adding Your Own Code

1. Write your code in C, Rust, or Go
2. Compile to WASM using the appropriate compiler
3. Test locally with `wasmer` or `wasmtime`
4. Upload and execute on the network

See `../multi_language_wasm_guide.md` for detailed instructions.
