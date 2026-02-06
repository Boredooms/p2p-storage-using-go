package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	// Read input from Stdin
	input, _ := io.ReadAll(os.Stdin)

	fmt.Printf("WASM Worker Info: Host OS is hidden.\n")
	fmt.Printf("Received Input: %s\n", string(input))
	fmt.Printf("Calculation Result: %d\n", len(input)*2)
}
