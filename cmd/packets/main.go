// Command packets runs the packets CLI.
package main

import (
	"fmt"
	"os"

	"github.com/joaomdsg/packets/internal/cli"
)

func main() {
	root := cli.NewRoot()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
