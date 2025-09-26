package commands

import "fmt"

// PrintUsage writes basic command help to stdout.
func PrintUsage() {
	fmt.Println("Usage: hyperctl <command> [options]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  setup      Bootstrap or update the hypervisor service")
	fmt.Println("  version    Show the currently installed hypervisor build")
	fmt.Println("  help       Show this help text")
}
