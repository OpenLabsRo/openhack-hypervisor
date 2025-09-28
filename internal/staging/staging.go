
package staging

import (
    "fmt"
    "hypervisor/internal/git"
    "hypervisor/internal/paths"
    "hypervisor/internal/releases/db"
    "os/exec"
)

const (
    repoURL = "https://github.com/openlabs-org/openhack-backend.git"
)

func StageRelease(tag string) error {
    repoPath := paths.OpenHackRepoPath(tag)

    // 1. Clone the specific tag
    if err := git.CloneTag(repoURL, tag, repoPath); err != nil {
        return err
    }

    // 2. Build the release
    buildScript := paths.OpenHackRepoPath(tag, "BUILD")
    buildCmd := exec.Command(buildScript)
    buildCmd.Dir = repoPath
    if output, err := buildCmd.CombinedOutput(); err != nil {
        return fmt.Errorf("build failed: %w\n%s", err, output)
    }

    // 3. Test the release
    testScript := paths.OpenHackRepoPath(tag, "TEST")
    testCmd := exec.Command(testScript)
    testCmd.Dir = repoPath
    if output, err := testCmd.CombinedOutput(); err != nil {
        return fmt.Errorf("test failed: %w\n%s", err, output)
    }

    // 4. Update release status in DB
    if err := db.UpdateStatus(tag, "staged"); err != nil {
        return err
    }

    // 5. Deploy to inactive slot (to be implemented)

    return nil
}
