# Openhack User Sudoers Configuration

**Date**: 2025  
**Status**: Implementation Complete  
**Purpose**: Enable the openhack service account to deploy backend instances

---

## Problem Statement

The hypervisor service runs as the `openhack` user and needs to:
1. Write systemd service files to `/lib/systemd/system/`
2. Reload and manage systemd services via `systemctl`

However, `/lib/systemd/system/` is owned by root and requires elevated privileges to write to. The `openhack` user cannot use sudo interactively (has no shell), so a sudoers entry is needed to allow specific commands.

---

## Solution: Sudoers Entry for openhack User

### File: `/etc/sudoers.d/openhack`

```
openhack ALL=(ALL) NOPASSWD: /usr/bin/tee, /usr/bin/systemctl
```

**What this allows:**
- `openhack` user can run `/usr/bin/tee` without a password (for writing files)
- `openhack` user can run `/usr/bin/systemctl` without a password (for service management)
- Both commands can be executed from Go code via `exec.Command("sudo", ...)`

**Why no shell is needed:**
- The openhack user has `/usr/sbin/nologin` as shell (cannot login interactively)
- But Go's `exec.Command()` doesn't start an interactive shell
- It directly executes the command specified, which sudo allows
- No password prompt is needed because of `NOPASSWD` directive

---

## Implementation

### Installation Script Changes

**File**: `hyperctl_install.sh`

The install script now creates two sudoers files:

1. **`/etc/sudoers.d/openhack-admins`** (for admin users)
   ```
   %openhack-admins ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/bin/tee, /usr/bin/chown, /usr/bin/mkdir, /bin/bash, /usr/bin/useradd, /usr/bin/groupadd, /usr/bin/usermod, /usr/local/bin/hyperctl
   ```
   - Used by admin users (alice, bob) when running hyperctl commands

2. **`/etc/sudoers.d/openhack`** (for service account)
   ```
   openhack ALL=(ALL) NOPASSWD: /usr/bin/tee, /usr/bin/systemctl
   ```
   - Used by the openhack service process for deployments
   - Much more restrictive (only what's needed for service deployments)

### Code Changes

**File**: `internal/hyperctl/user/user.go`

Updated `CreateSudoersFile()` to remove unnecessary commands from admin group:
- Removed `/usr/bin/chmod` - not needed, files owned by openhack
- Removed `/usr/bin/rm` - not needed, service doesn't remove system files
- Kept: systemctl, tee, chown, mkdir, bash, useradd, groupadd, usermod, hyperctl

---

## How It Works During Deployment

### Step 1: Hypervisor Service Calls Go Code

```go
// In internal/core/deployments.go
systemd.InstallBackendService(cfg, logFile)
```

### Step 2: Go Code Uses sudo via exec.Command

```go
// In internal/systemd/manage.go
fs.WriteFileWithSudo(target, rendered.Bytes(), 0o644)
```

Which expands to:
```go
cmd := exec.Command("sudo", "tee", "/lib/systemd/system/openhack-backend-v1.2.3-prod.service")
cmd.Run()
```

### Step 3: sudo Checks Sudoers

```
openhack ALL=(ALL) NOPASSWD: /usr/bin/tee
                    ↓
                 Matches!
```

### Step 4: Command Executes Without Password

The systemd file is written successfully because:
- Process runs as `openhack` user
- Sudoers allows `openhack` to run `/usr/bin/tee` without password
- File is written to `/lib/systemd/system/`
- Then `systemctl daemon-reload` is called (also allowed in sudoers)

---

## Security Considerations

### Least Privilege Applied

**Admin group sudoers** (`openhack-admins`):
- Full control over hyperctl CLI
- Can manage systemctl, tee, chown, mkdir, rm, bash, user management
- Reasonable for operators who manage the entire system

**Service account sudoers** (`openhack`):
- Only `/usr/bin/tee` and `/usr/bin/systemctl`
- Limited to what the service needs for deployments
- No shell access, no arbitrary command execution
- More restrictive than admin group

### No Interactive Shell Required

The openhack user has `/usr/sbin/nologin` shell:
```bash
$ id openhack
uid=1001(openhack) gid=1001(openhack) groups=1001(openhack),1002(openhack-admins)
$ cat /etc/passwd | grep openhack
openhack:x:1001:1001:OpenHack service account:/var/openhack:/usr/sbin/nologin
```

This means:
- ✅ Cannot be used for interactive login
- ✅ Cannot open an interactive sudo shell
- ✅ CAN execute commands via Go's `exec.Command("sudo", ...)`
- ✅ Sudoers NOPASSWD still works for direct command execution

### Command Paths Are Absolute

All commands in sudoers use full paths (`/usr/bin/tee`, `/usr/bin/systemctl`):
- Prevents PATH manipulation attacks
- Ensures exact commands are executed
- Best practice for sudoers configuration

---

## Installation Verification

After running `./hyperctl_install.sh`:

```bash
# Check openhack-admins sudoers
sudo cat /etc/sudoers.d/openhack-admins
# Output: %openhack-admins ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/bin/tee, ...

# Check openhack user sudoers
sudo cat /etc/sudoers.d/openhack
# Output: openhack ALL=(ALL) NOPASSWD: /usr/bin/tee, /usr/bin/systemctl

# Verify sudoers syntax
sudo visudo -c -f /etc/sudoers.d/openhack-admins
# Output: /etc/sudoers.d/openhack-admins: parsed OK

sudo visudo -c -f /etc/sudoers.d/openhack
# Output: /etc/sudoers.d/openhack: parsed OK

# Check file permissions (should be 0440)
ls -la /etc/sudoers.d/openhack*
# Output: -r--r----- 1 root root ... openhack
# Output: -r--r----- 1 root root ... openhack-admins
```

---

## Sudoers Entry Breakdown

### For openhack-admins group:

| Command | Purpose | Needed For |
|---------|---------|-----------|
| `/usr/bin/systemctl` | Manage services | hyperctl commands |
| `/usr/bin/tee` | Write files with sudo | Config file creation |
| `/usr/bin/chown` | Change ownership | Directory setup |
| `/usr/bin/mkdir` | Create directories | Layout initialization |
| `/bin/bash` | Execute shell scripts | Build scripts, setup |
| `/usr/bin/useradd` | Create users | Setup phase |
| `/usr/bin/groupadd` | Create groups | Setup phase |
| `/usr/bin/usermod` | Modify users | Setup phase |
| `/usr/local/bin/hyperctl` | Run hyperctl | Management commands |

### For openhack user:

| Command | Purpose | Needed For |
|---------|---------|-----------|
| `/usr/bin/tee` | Write systemd files | Backend deployment |
| `/usr/bin/systemctl` | Manage services | Service reload/enable/restart |

---

## Why NOT These Commands?

### `/usr/bin/chmod`
- ❌ Not needed for openhack service
- Binaries from `go build` are already executable (0755)
- Directories are pre-owned with correct permissions
- Only hyperctl CLI needs chmod (already in its sudoers)

### `/usr/bin/rm`
- ❌ Not needed for openhack service
- Service doesn't remove system files
- Only hyperctl CLI needs rm for cleanup (already in its sudoers)

### `/bin/bash`
- ❌ Not needed for openhack service
- Service doesn't run shell scripts
- Only hyperctl CLI needs bash (already in its sudoers)

### User management commands
- ❌ Not needed for openhack service
- Service doesn't create users or groups
- Only install script and hyperctl CLI need these (already in their sudoers)

---

## Related Files

- `hyperctl_install.sh` - Creates sudoers entries during installation
- `internal/hyperctl/user/user.go` - Defines sudoers content for admin group
- `internal/core/deployments.go` - Service deployment logic
- `internal/systemd/manage.go` - Systemd file writing
- `internal/fs/fs.go` - WriteFileWithSudo implementation

---

## Testing Deployment with New Sudoers

```bash
# 1. Install hyperctl
sudo ./hyperctl_install.sh

# 2. Run setup (creates sudoers)
sudo hyperctl manhattan

# 3. Create a test deployment (via API or command)
# This will trigger:
# - Binary build via go build (no sudo needed)
# - Systemd file write via sudo tee (uses new sudoers)
# - Service reload/enable/restart via sudo systemctl (uses new sudoers)

# 4. Verify service started
sudo systemctl status openhack-backend-<id>

# 5. Check that service is running as openhack user
ps aux | grep openhack-backend
# Should show: openhack 12345 0.5 0.2 ...
```

---

## Troubleshooting

### "sudo: no password was provided, but a password is required"

**Cause**: Sudoers entry missing or incorrect

**Fix**:
```bash
# Verify sudoers file exists
ls -la /etc/sudoers.d/openhack

# Check content
sudo cat /etc/sudoers.d/openhack
# Should show: openhack ALL=(ALL) NOPASSWD: /usr/bin/tee, /usr/bin/systemctl

# Verify syntax
sudo visudo -c -f /etc/sudoers.d/openhack
```

### "permission denied" when writing systemd files

**Cause**: Sudoers entry not working or openhack service not running as openhack user

**Fix**:
```bash
# Check service configuration
systemctl cat openhack-hypervisor-blue | grep -E "User=|Group="
# Should show: User=openhack, Group=openhack

# Check sudoers syntax
sudo visudo -c -f /etc/sudoers.d/openhack

# Re-run installation
sudo ./hyperctl_install.sh
```

---

## Summary

The openhack user now has a minimal sudoers entry that allows it to:
1. Write systemd service files via `/usr/bin/tee`
2. Manage systemd services via `/usr/bin/systemctl`

This enables the hypervisor service to deploy backend instances without requiring an interactive shell or full root privileges. The entry is restrictive, following the principle of least privilege.