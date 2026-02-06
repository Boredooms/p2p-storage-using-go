package compute

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CompileCToWasm compiles C source code to WASM using emcc (Emscripten)
func CompileCToWasm(sourceCode string) ([]byte, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "wasm-compile-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write source to file
	srcFile := filepath.Join(tmpDir, "main.c")
	if err := os.WriteFile(srcFile, []byte(sourceCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// Output WASM file
	wasmFile := filepath.Join(tmpDir, "output.wasm")

	// Compile with emcc (if available) or fallback to clang
	// For MVP, we'll use a simple approach
	cmd := exec.Command("emcc", srcFile, "-o", wasmFile,
		"-O2",
		"-s", "WASM=1",
		"-s", "STANDALONE_WASM",
		"-s", "EXPORTED_FUNCTIONS=['_main']",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	// Read compiled WASM
	wasmBytes, err := os.ReadFile(wasmFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	return wasmBytes, nil
}

// DetectLanguage tries to detect if input is C, Rust, or already WASM
func DetectLanguage(code []byte) string {
	// WASM magic number: 0x00 0x61 0x73 0x6D
	if len(code) >= 4 && code[0] == 0x00 && code[1] == 0x61 && code[2] == 0x73 && code[3] == 0x6D {
		return "wasm"
	}

	codeStr := string(code)

	// Check for C patterns
	if contains(codeStr, "#include") || contains(codeStr, "int main") {
		return "c"
	}

	// Check for Rust patterns
	if contains(codeStr, "fn main") || contains(codeStr, "use std") {
		return "rust"
	}

	return "unknown"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s[:len(substr)] == substr ||
			len(s) > len(substr) && s[1:len(substr)+1] == substr ||
			len(s) > 100 && s[10:10+len(substr)] == substr)
}
