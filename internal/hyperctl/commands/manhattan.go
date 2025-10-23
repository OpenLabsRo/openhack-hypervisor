package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/git"
	"hypervisor/internal/hyperctl/health"
	"hypervisor/internal/hyperctl/state"
	"hypervisor/internal/hyperctl/system"
	"hypervisor/internal/hyperctl/systemd"
	userutil "hypervisor/internal/hyperctl/user"
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

	// Verify openhack user and group exist (created by install script)
	fmt.Println("Verifying openhack user and group...")
	if !userutil.UserExists(userutil.OpenhackUser) {
		return fmt.Errorf("openhack user not found - run hyperctl_install.sh first to set up the openhack user and group")
	}
	if !userutil.GroupExists(userutil.OpenhackAdminGroup) {
		return fmt.Errorf("openhack-admins group not found - run hyperctl_install.sh first to set up the openhack user and group")
	}
	fmt.Println("openhack user and group verified")

	// Check if calling user is in openhack-admins group
	adminUser, err := userutil.GetAdminUser()
	if err != nil {
		return fmt.Errorf("failed to determine admin user: %w", err)
	}
	if adminUser != userutil.OpenhackUser && adminUser != "root" {
		inGroup, err := userutil.IsInGroup(adminUser, userutil.OpenhackAdminGroup)
		if err != nil {
			fmt.Printf("Warning: could not verify %s is in %s group: %v\n", adminUser, userutil.OpenhackAdminGroup, err)
			fmt.Printf("Tip: Run 'newgrp %s' in current shell or log out and back in\n", userutil.OpenhackAdminGroup)
		} else if !inGroup {
			fmt.Printf("Warning: %s is not in %s group\n", adminUser, userutil.OpenhackAdminGroup)
			fmt.Printf("Tip: Run hyperctl_install.sh again to add %s to the group, then log out and back in\n", adminUser)
		}
	}

	// Verify system requirements
	if err := system.CheckPrerequisites(); err != nil {
		return err
	}

	// Create necessary directories (ensures proper ownership/permissions)
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

	// Ensure swagger docs directory exists before running API_SPEC
	swaggerDocsDir := filepath.Join(repoDir, "internal/swagger/docs")
	if err := os.MkdirAll(swaggerDocsDir, 0755); err != nil {
		return fmt.Errorf("failed to create swagger docs directory: %w", err)
	}
	fmt.Printf("Swagger docs directory ensured at %s\n", swaggerDocsDir)

	// Run API_SPEC
	cmd := exec.Command("./API_SPEC")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run API_SPEC: %w", err)
	}
	fmt.Println("API_SPEC executed successfully")

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
