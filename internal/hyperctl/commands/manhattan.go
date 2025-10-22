package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/git"
	"hypervisor/internal/hyperctl/health"
	"hypervisor/internal/hyperctl/state"
	"hypervisor/internal/hyperctl/system"
	"hypervisor/internal/hyperctl/systemd"
	"hypervisor/internal/paths"
)

const (
	hypervisorRepoURL = "https://github.com/OpenLabsRo/openhack-hypervisor"
)

// RunManhattan handles the `hyperctl manhattan` subcommand.
func RunManhattan(args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet("manhattan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dev := fs.Bool("dev", false, "Development mode: use current directory instead of cloning")
	moveEnv := fs.Bool("move", false, "Copy .env files from local dirs to system dirs")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Starting hypervisor setup...")

	// Verify system requirements
	if err := system.CheckPrerequisites(); err != nil {
		return err
	}

	// Create necessary directories
	fmt.Println("Ensuring directory layout...")
	if err := fsops.EnsureLayout(); err != nil {
		return err
	}
	fmt.Println("Directory layout ready")

	// Copy .env files if requested
	if *moveEnv {
		fmt.Println("Copying .env files...")
		home := os.Getenv("HOME")
		src1 := filepath.Join(home, "coding/openlabs/openhack-hypervisor/.env")
		dest1 := "/var/hypervisor/env/.env"
		if err := os.MkdirAll(filepath.Dir(dest1), 0755); err != nil {
			fmt.Printf("Warning: Failed to create dir for %s: %v\n", dest1, err)
		}
		if err := copyFile(src1, dest1); err != nil {
			fmt.Printf("Warning: Failed to copy %s to %s: %v\n", src1, dest1, err)
		} else {
			fmt.Printf("Copied %s to %s\n", src1, dest1)
		}

		src2 := filepath.Join(home, "coding/openlabs/openhack-backend/.env")
		dest2 := "/var/openhack/env/template/.env"
		if err := os.MkdirAll(filepath.Dir(dest2), 0755); err != nil {
			fmt.Printf("Warning: Failed to create dir for %s: %v\n", dest2, err)
		}
		if err := copyFile(src2, dest2); err != nil {
			fmt.Printf("Warning: Failed to copy %s to %s: %v\n", src2, dest2, err)
		} else {
			fmt.Printf("Copied %s to %s\n", src2, dest2)
		}
	}

	// Configure environment files
	editor := system.ResolveEditor()

	hypervisorEnvPath := fsops.HypervisorEnvPath()
	if err := fsops.EnsureEnvDirFor(hypervisorEnvPath); err != nil {
		return err
	}
	if err := fsops.EditEnvFileIfMissing("Hypervisor", hypervisorEnvPath, editor); err != nil {
		return err
	}

	openhackEnvTemplatePath := fsops.OpenHackEnvTemplatePath()
	if err := fsops.EnsureEnvDirFor(openhackEnvTemplatePath); err != nil {
		return err
	}
	if err := fsops.EditEnvFileIfMissing("OpenHack Backend Template", openhackEnvTemplatePath, editor); err != nil {
		return err
	}

	// Clone or use local project repository
	fmt.Println("Cloning the project...")
	var repoDir string
	if *dev {
		repoDir = "."
		fmt.Println("Development mode: using current directory")
	} else {
		repoDir = filepath.Join(paths.HypervisorReposDir, "main")
		if err := git.CloneOrPull(hypervisorRepoURL, repoDir); err != nil {
			return fmt.Errorf("failed to clone project: %w", err)
		}
		fmt.Printf("Project cloned to %s\n", repoDir)
	}

	// Run API_SPEC
	cmd := exec.Command("./API_SPEC")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run API_SPEC: %w", err)
	}
	fmt.Println("API_SPEC executed successfully")

	// Tests removed

	// Build the application
	fmt.Printf("Building the code into %s...\n", paths.HypervisorBuildsDir)
	buildResult, err := build.Run(repoDir, paths.HypervisorBuildsDir)
	if err != nil {
		return err
	}
	fmt.Printf("Build complete: %s\n", buildResult.BinaryPath)

	// Install and configure systemd services (blue and green)
	fmt.Println("Installing systemd units for hypervisor (blue and green)...")

	deployment := "prod"
	if *dev {
		deployment = "dev"
	}

	// Blue service (port 8080)
	blueCfg := systemd.ServiceConfig{
		BinaryPath: buildResult.BinaryPath,
		Deployment: deployment,
		Port:       "8080",
		EnvRoot:    paths.HypervisorEnvDir,
		Version:    buildResult.Version,
	}
	if err := systemd.InstallHypervisorService(blueCfg, "blue"); err != nil {
		return err
	}

	// Green service (port 8081)
	greenCfg := systemd.ServiceConfig{
		BinaryPath: buildResult.BinaryPath,
		Deployment: deployment,
		Port:       "8081",
		EnvRoot:    paths.HypervisorEnvDir,
		Version:    buildResult.Version,
	}
	if err := systemd.InstallHypervisorService(greenCfg, "green"); err != nil {
		return err
	}

	fmt.Println("Systemd units installed (blue on 8080, green on 8081)")

	// Persist installation state
	fmt.Println("Persisting installation state...")
	if err := state.Save(state.State{
		Version:   buildResult.Version,
		BuildPath: buildResult.BinaryPath,
	}); err != nil {
		return err
	}
	fmt.Println("State saved")

	// Final health verification
	fmt.Println("Checking service status...")
	if err := systemd.CheckServiceStatus("blue"); err != nil {
		return fmt.Errorf("blue service status check failed: %w", err)
	}
	if err := systemd.CheckServiceStatus("green"); err != nil {
		return fmt.Errorf("green service status check failed: %w", err)
	}
	fmt.Println("Services are active (blue on 8080, green on 8081)")

	// Configure sudoers for passwordless commands
	fmt.Println("Configuring sudoers for passwordless commands...")
	if err := configureSudoers(); err != nil {
		return fmt.Errorf("failed to configure sudoers: %w", err)
	}
	fmt.Println("Sudoers configured")

	// Final health verification
	fmt.Println("Performing health check...")
	if err := health.CheckHost("localhost:8080"); err != nil {
		return fmt.Errorf("blue service health check failed: %w", err)
	}
	if err := health.CheckHost("localhost:8081"); err != nil {
		return fmt.Errorf("green service health check failed: %w", err)
	}
	fmt.Println("Health checks passed")

	fmt.Println("Setup completed successfully.")

	return nil
}

func copyFile(src, dest string) error {
	cmd := exec.Command("sudo", "cp", src, dest)
	return cmd.Run()
}

// configureSudoers adds a sudoers file for passwordless commands.
func configureSudoers() error {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		// Fallback to current user if not run with sudo
		if u, err := user.Current(); err == nil {
			sudoUser = u.Username
		} else {
			return fmt.Errorf("unable to determine user")
		}
	}

	content := fmt.Sprintf(`%s ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/bin/tee, /usr/bin/chmod, /usr/bin/rm
`, sudoUser)

	sudoersFile := "/etc/sudoers.d/hypervisor"
	if err := fsops.WriteFileWithSudo(sudoersFile, []byte(content), 0o440); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	return nil
}
