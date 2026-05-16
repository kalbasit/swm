// fakefzf is a fake fzf binary used in unit tests.
// It reads lines from stdin and outputs the first one to stdout.
// Set FAKEFZF_EXIT=1 to simulate the user cancelling (exit code 1).
package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	if os.Getenv("FAKEFZF_EXIT") == "1" {
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}
