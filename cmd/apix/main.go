// Package main is the entry point for the apix CLI.
package main

import (
	"fmt"
	"os"

	"github.com/Tresor-Kasend/apix/internal/cli"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
