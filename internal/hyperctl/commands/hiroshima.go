package commands

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/systemd"
	"hypervisor/internal/paths"
)

// RunHiroshima handles the `hyperctl hiroshima` subcommand.
// This command completely removes the directories created by hyperctl setup.
func RunHiroshima(args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet("hiroshima", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	// Optional flag to confirm destruction
	confirm := fs.Bool("yes", false, "Confirm the destructive operation without prompting")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Safety confirmation
	if !*confirm {
		fmt.Println("WARNING: This command will completely remove /var/hypervisor and /var/openhack directories.")
		fmt.Println("This action is irreversible and will delete all data, configurations, and builds.")
		fmt.Print("Are you sure? Type 'yes' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Stop the running service
	fmt.Println("Stopping hypervisor service...")
	if err := systemd.StopHypervisorService(); err != nil {
		fmt.Printf("Warning: Failed to stop service: %v\n", err)
	}

	// Disable the service
	fmt.Println("Disabling hypervisor service...")
	if err := systemd.DisableHypervisorService(); err != nil {
		fmt.Printf("Warning: Failed to disable service: %v\n", err)
	}

	// Stop blue and green services
	fmt.Println("Stopping blue and green services...")
	if err := systemd.StopService("openhack-hypervisor-blue.service"); err != nil {
		fmt.Printf("Warning: Failed to stop blue service: %v\n", err)
	}
	if err := systemd.StopService("openhack-hypervisor-green.service"); err != nil {
		fmt.Printf("Warning: Failed to stop green service: %v\n", err)
	}

	// Remove the systemd service file
	fmt.Println("Removing systemd service file...")
	if err := systemd.RemoveServiceFile(); err != nil {
		fmt.Printf("Warning: Failed to remove service file: %v\n", err)
	}

	// Stop all backend services
	fmt.Println("Stopping all backend services...")
	stopCmd := exec.Command("bash", "-c", "systemctl list-units --no-legend --state=active,failed | grep openhack-backend | awk '{print $1}' | xargs -r systemctl stop")
	if output, err := stopCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Failed to stop some backend services: %v\n%s\n", err, string(output))
	} else {
		fmt.Println("Backend services stopped.")
	}

	// Reload systemd to apply changes
	fmt.Println("Reloading systemd...")
	if err := systemd.ReloadSystemd(); err != nil {
		fmt.Printf("Warning: Failed to reload systemd: %v\n", err)
	}

	// Remove all created directories
	fmt.Println("Removing /var/hypervisor...")
	if err := fsops.RemoveDir(paths.HypervisorBaseDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", paths.HypervisorBaseDir, err)
	}

	fmt.Println("Removing /var/openhack...")
	if err := fsops.RemoveDir(paths.OpenHackBaseDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", paths.OpenHackBaseDir, err)
	}

	fmt.Println("Uninstallation complete. All hypervisor and OpenHack data has been removed.")
	return nil
}
