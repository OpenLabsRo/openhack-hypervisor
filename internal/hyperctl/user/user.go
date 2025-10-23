package user

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

const (
	OpenhackUser       = "openhack"
	OpenhackAdminGroup = "openhack-admins"
)

// OpenhackUID returns the UID of the openhack user, or an error if not found.
func OpenhackUID() (int, error) {
	u, err := user.Lookup(OpenhackUser)
	if err != nil {
		return 0, fmt.Errorf("openhack user not found: %w", err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("invalid openhack UID: %w", err)
	}
	return uid, nil
}

// OpenhackAdminGID returns the GID of the openhack-admins group, or an error if not found.
func OpenhackAdminGID() (int, error) {
	g, err := user.LookupGroup(OpenhackAdminGroup)
	if err != nil {
		return 0, fmt.Errorf("%s group not found: %w", OpenhackAdminGroup, err)
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, fmt.Errorf("invalid %s GID: %w", OpenhackAdminGroup, err)
	}
	return gid, nil
}

// OpenhackOwner returns the "uid:gid" string for chown operations.
func OpenhackOwner() (string, error) {
	uid, err := OpenhackUID()
	if err != nil {
		return "", err
	}
	gid, err := OpenhackAdminGID()
	if err != nil {
		// If openhack-admins group doesn't exist yet, fall back to openhack's primary group
		u, err2 := user.Lookup(OpenhackUser)
		if err2 != nil {
			return "", err
		}
		gid, err2 = strconv.Atoi(u.Gid)
		if err2 != nil {
			return "", fmt.Errorf("invalid openhack primary GID: %w", err2)
		}
	}
	return fmt.Sprintf("%d:%d", uid, gid), nil
}

// ChownToOpenhack recursively changes ownership of a path to openhack user/group.
func ChownToOpenhack(path string) error {
	owner, err := OpenhackOwner()
	if err != nil {
		return err
	}

	cmd := exec.Command("sudo", "chown", "-R", owner, path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo chown -R %s %s: %w", owner, path, err)
	}
	return nil
}

// ChmodPath changes permissions on a path.
func ChmodPath(path string, perm string) error {
	cmd := exec.Command("sudo", "chmod", perm, path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo chmod %s %s: %w", perm, path, err)
	}
	return nil
}

// IsOpenhackUser checks if the current user is the openhack user.
func IsOpenhackUser() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	return currentUser.Username == OpenhackUser
}

// GetEffectiveUser returns the actual user executing the command (handles sudo).
func GetEffectiveUser() (string, error) {
	// If run with sudo, get the original user
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		return sudoUser, nil
	}

	// Otherwise get current user
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to determine current user: %w", err)
	}
	return currentUser.Username, nil
}

// GetAdminUser returns the effective admin user (handles sudo context).
func GetAdminUser() (string, error) {
	// If run with sudo, use the original user
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		return sudoUser, nil
	}

	// Fallback to current user
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to determine admin user: %w", err)
	}
	return currentUser.Username, nil
}

// UserExists checks if a given user exists on the system.
func UserExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

// GroupExists checks if a given group exists on the system.
func GroupExists(groupname string) bool {
	_, err := user.LookupGroup(groupname)
	return err == nil
}

// CreateUserIfNotExists creates the openhack system user if it doesn't exist.
func CreateUserIfNotExists() error {
	if UserExists(OpenhackUser) {
		fmt.Printf("openhack user already exists\n")
		return nil
	}

	fmt.Printf("Creating openhack system user...\n")
	cmd := exec.Command("sudo", "useradd",
		"--system",
		"--home-dir", "/var/openhack",
		"--shell", "/usr/sbin/nologin",
		OpenhackUser,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create openhack user: %w", err)
	}
	fmt.Printf("openhack user created\n")
	return nil
}

// CreateGroupIfNotExists creates the openhack-admins group if it doesn't exist.
func CreateGroupIfNotExists() error {
	if GroupExists(OpenhackAdminGroup) {
		fmt.Printf("%s group already exists\n", OpenhackAdminGroup)
		return nil
	}

	fmt.Printf("Creating %s group...\n", OpenhackAdminGroup)
	cmd := exec.Command("sudo", "groupadd", OpenhackAdminGroup)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create %s group: %w", OpenhackAdminGroup, err)
	}
	fmt.Printf("%s group created\n", OpenhackAdminGroup)
	return nil
}

// AddUserToGroup adds a user to a group.
func AddUserToGroup(username, groupname string) error {
	fmt.Printf("Adding %s to %s group...\n", username, groupname)
	cmd := exec.Command("sudo", "usermod", "-a", "-G", groupname, username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add %s to %s group: %w", username, groupname, err)
	}
	fmt.Printf("%s added to %s group\n", username, groupname)
	return nil
}

// IsInGroup checks if a user is in a group.
func IsInGroup(username, groupname string) (bool, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	groups, err := u.GroupIds()
	if err != nil {
		return false, fmt.Errorf("failed to get user groups: %w", err)
	}

	g, err := user.LookupGroup(groupname)
	if err != nil {
		return false, fmt.Errorf("group not found: %w", err)
	}

	for _, gid := range groups {
		if gid == g.Gid {
			return true, nil
		}
	}
	return false, nil
}

// CreateSudoersFile creates a sudoers file for the openhack-admins group.
func CreateSudoersFile() error {
	content := fmt.Sprintf(`%%%s ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/bin/tee, /usr/bin/chown, /usr/bin/mkdir, /bin/bash, /usr/bin/useradd, /usr/bin/groupadd, /usr/bin/usermod, /usr/local/bin/hyperctl
`, OpenhackAdminGroup)

	sudoersFile := "/etc/sudoers.d/openhack-admins"

	fmt.Printf("Writing sudoers file at %s...\n", sudoersFile)

	// Write file using sudo tee
	cmd := exec.Command("sudo", "tee", sudoersFile)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	// Set correct permissions (0440)
	if err := ChmodPath(sudoersFile, "0440"); err != nil {
		return fmt.Errorf("failed to chmod sudoers file: %w", err)
	}

	fmt.Printf("Sudoers file created at %s\n", sudoersFile)
	return nil
}
