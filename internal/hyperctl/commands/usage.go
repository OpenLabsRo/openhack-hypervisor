package commands

import "fmt"

// PrintUsage writes basic command help to stdout.
func PrintUsage() {
	fmt.Println("Usage: hyperctl <command> [options]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  manhattan  Bootstrap or update the hypervisor service")
	fmt.Println("  nagasaki   Stop the running hypervisor service")
	fmt.Println("  hiroshima  Completely remove all hypervisor and OpenHack directories (destructive)")
	fmt.Println("  ping       Ping the hypervisor health endpoint")
	fmt.Println("  interstate Show the current routing map")
	fmt.Println("  trinity    Update hypervisor by cloning, testing, and building new version")
	fmt.Println("  swaddle    Generate nginx configuration for the hypervisor")
	fmt.Println("  grimhilde  Update hyperctl to the latest version")
	fmt.Println("  version    Show the currently installed hypervisor build")
	fmt.Println("  help       Show this help text")
}
