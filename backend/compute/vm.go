package compute

import (
	"bytes"
	"context"
	"fmt"


	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// VM handles safe execution of WebAssembly code.
type VM struct {
	runtime wazero.Runtime
	ctx     context.Context
}

func NewVM(ctx context.Context) *VM {
	// Create a new WebAssembly Runtime.
	r := wazero.NewRuntime(ctx)
	
	// Instantiate WASI (WebAssembly System Interface) so modules can use basic I/O (print, etc).
	// We bind it effectively allowing limited access.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &VM{
		runtime: r,
		ctx:     ctx,
	}
}

// Run executes a WASM binary with the given input data passed to stdin.
// Returns the stdout output.
func (v *VM) Run(wasmCode []byte, inputData []byte) ([]byte, error) {
	// Compile the module.
	compiled, err := v.runtime.CompileModule(v.ctx, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("failed to compile wasm: %w", err)
	}
	defer compiled.Close(v.ctx)

	// Config for the module instance.
	// We capture stdout/stderr to return results.
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	inputBuf := bytes.NewReader(inputData)

	config := wazero.NewModuleConfig().
		WithStdin(inputBuf).
		WithStdout(&stdoutBuf).
		WithStderr(&stderrBuf).
		// SECURITY: We do NOT mount any real verification filesystem.
		// The guest module runs in a void.
		WithArgs("job") 

	// Instantiate and Run.
	// This runs the "_start" function by default (like main() in C/Go).
	mod, err := v.runtime.InstantiateModule(v.ctx, compiled, config)
	if err != nil {
		return nil, fmt.Errorf("runtime error: %w (stderr: %s)", err, stderrBuf.String())
	}
	defer mod.Close(v.ctx)

	// Since _start implies the program finished, we just return the captured stdout.
	return stdoutBuf.Bytes(), nil
}

func (v *VM) Close() error {
	return v.runtime.Close(v.ctx)
}

// Echo is a test function to verify the VM works
func (v *VM) CheckHealth() bool {
	return v.runtime != nil
}
