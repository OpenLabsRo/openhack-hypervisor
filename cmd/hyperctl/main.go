package main

import (
	"fmt"
	"os"

	"hypervisor/internal/hyperctl/commands"
)

func main() {
	// Ensure hyperctl is always run with sudo
	if os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, "hyperctl: this command must be run with sudo\n")
		os.Exit(1)
	}

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
	case "manhattan":
		err = commands.RunManhattan(args)
	case "hiroshima":
		err = commands.RunHiroshima(args)
	case "nagasaki":
		err = commands.RunNagasaki(args)
	case "ping":
		err = commands.RunPing(args)
	case "interstate":
		err = commands.RunInterstate(args)
	case "trinity":
		err = commands.RunTrinity(args)
	case "swaddle":
		err = commands.RunSwaddle(args)
	case "grimhilde":
		err = commands.RunGrimhilde(args)
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
