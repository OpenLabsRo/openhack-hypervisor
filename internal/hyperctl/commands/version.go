package commands

import (
	"flag"
	"fmt"
	"io"

	"hypervisor/internal/hyperctl/state"
)

// RunVersion handles the `hyperctl version` subcommand.
func RunVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}

	version, err := state.CurrentVersion()
	if err != nil {
		if err == state.ErrStateNotInitialized {
			fmt.Println("No hypervisor build installed. Run `hyperctl setup` first.")
			return nil
		}
		return err
	}

	fmt.Println(version)
	return nil
}
