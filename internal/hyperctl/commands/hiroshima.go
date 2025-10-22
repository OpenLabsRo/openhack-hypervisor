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

	// Stop blue and green services
	fmt.Println("Stopping blue and green services...")
	if err := systemd.StopService("openhack-hypervisor-blue.service"); err != nil {
		fmt.Printf("Warning: Failed to stop blue service: %v\n", err)
	}
	if err := systemd.StopService("openhack-hypervisor-green.service"); err != nil {
		fmt.Printf("Warning: Failed to stop green service: %v\n", err)
	}

	// Stop all backend services
	fmt.Println("Stopping all backend services...")
	stopCmd := exec.Command("bash", "-c", "systemctl list-units --no-legend --state=active,failed | grep openhack-backend | awk '{print $1}' | xargs -r sudo systemctl stop")
	if output, err := stopCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Failed to stop some backend services: %v\n%s\n", err, string(output))
	} else {
		fmt.Println("Backend services stopped.")
	}

	// Remove backend service unit files
	fmt.Println("Removing backend service unit files...")
	removeBackendCmd := exec.Command("sudo", "rm", "-f", "/lib/systemd/system/openhack-backend-*.service")
	if output, err := removeBackendCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Failed to remove backend service files: %v\n%s\n", err, string(output))
	} else {
		fmt.Println("Backend service files removed.")
	}

	// Disable and remove blue/green hypervisor services
	fmt.Println("Disabling and removing blue/green hypervisor services...")
	disableBlueCmd := exec.Command("sudo", "systemctl", "disable", "openhack-hypervisor-blue.service")
	if err := disableBlueCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to disable blue service: %v\n", err)
	}
	disableGreenCmd := exec.Command("sudo", "systemctl", "disable", "openhack-hypervisor-green.service")
	if err := disableGreenCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to disable green service: %v\n", err)
	}
	removeBlueCmd := exec.Command("sudo", "rm", "-f", "/lib/systemd/system/openhack-hypervisor-blue.service")
	if err := removeBlueCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to remove blue service file: %v\n", err)
	}
	removeGreenCmd := exec.Command("sudo", "rm", "-f", "/lib/systemd/system/openhack-hypervisor-green.service")
	if err := removeGreenCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to remove green service file: %v\n", err)
	}

	// Reload systemd daemon
	fmt.Println("Reloading systemd daemon...")
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
