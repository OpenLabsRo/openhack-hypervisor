package main

import (
	"fmt"
	"os"

	"hypervisor/internal/hyperctl/commands"
)

func main() {
	if len(os.Args) < 2 {
		commands.PrintUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error

	switch cmd {
	case "help", "--help", "-h":
		commands.PrintUsage()
		return
	case "version":
		err = commands.RunVersion(args)
	case "setup":
		err = commands.RunSetup(args)
	case "hiroshima":
		err = commands.RunHiroshima(args)
	case "nagasaki":
		err = commands.RunNagasaki(args)
	default:
		fmt.Fprintf(os.Stderr, "hyperctl: unknown command %q\n", cmd)
		commands.PrintUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "hyperctl %s: %v\n", cmd, err)
		os.Exit(1)
	}
}
