package main

import (
	"fmt"
	"os"

	"github.com/gitsoecode/increasex-cli-mcp/internal/cli"
)

func main() {
	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		msg := cli.FormatCLIError(err)
		if msg == "" {
			msg = err.Error()
		}
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

