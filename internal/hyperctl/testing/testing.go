package testing

import (
	"fmt"
	"os"
	"os/exec"

	"hypervisor/internal/paths"
)

// RunTests executes the project's test suite.
func RunTests(repoDir string) error {
	fmt.Println("========== RUNNING TESTS ==========")
	err := runTestScript(repoDir)
	if err != nil {
		fmt.Println("========== TESTS FAILED ==========")
		return err
	}
	fmt.Println("========== TESTS PASSED ==========")
	return nil
}

func runTestScript(repoDir string) error {
	cmd := exec.Command("./TEST", "--env-root", paths.HypervisorEnvDir)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
