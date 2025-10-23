#!/usr/bin/env sh

OWNER=openlabsro
REPO=openhack-hypervisor
ASSET_NAME=hyperctl
BINARY_NAME=hyperctl
INSTALL_DIR=/usr/local/bin
DOWNLOAD_URL="https://github.com/${OWNER}/${REPO}/releases/latest/download/${ASSET_NAME}"

OPENHACK_USER="openhack"
OPENHACK_ADMIN_GROUP="openhack-admins"
OPENHACK_HOME="/var/openhack"

err() {
  printf "hyperctl installer error: %s\n" "$1" >&2
}

die() {
  err "$1"
  exit 1
}

usage() {
  cat <<USAGE
Usage: ./hyperctl_install.sh [--nodeps|-nodeps|--fedora-deps]

Installs the hyperctl binary (and, by default, its prerequisites:
Go 1.25.1, redis-server, nginx, certbot integration, vim, and the swag CLI).

Also creates the openhack system user and openhack-admins group for managing
hypervisor operations, and adds the invoking user to the admin group.

Options:
  --nodeps, -nodeps      Skip installing prerequisite packages and toolchains.
  --fedora-deps          Install prerequisites for Fedora/RHEL-based systems (uses dnf).
  -h, --help             Show this help text.
USAGE
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

run_privileged() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    die "This action requires root privileges. Re-run as root or install dependencies manually with --nodeps."
  fi
}

append_path_entry() {
  ENTRY=$1

  case ":${PATH}:" in
    *":${ENTRY}:"*) ;;
    *) PATH="${PATH}:${ENTRY}"
       export PATH
       ;;
  esac

  PROFILE_PATH="${HOME}/.profile"

  if [ ! -e "$PROFILE_PATH" ]; then
    if ! touch "$PROFILE_PATH" 2>/dev/null; then
      printf "Warning: unable to create %s to persist PATH entry %s\n" "$PROFILE_PATH" "$ENTRY"
      return
    fi
  fi

  if [ -w "$PROFILE_PATH" ]; then
    if ! grep -qs "$ENTRY" "$PROFILE_PATH"; then
      printf 'export PATH="$PATH:%s"\n' "$ENTRY" >>"$PROFILE_PATH"
      printf "Added %s to PATH in %s\n" "$ENTRY" "$PROFILE_PATH"
    fi
  else
    printf "Warning: %s is not writable; ensure %s is on PATH manually.\n" "$PROFILE_PATH" "$ENTRY"
  fi

  source ~/.profile
}

install_go_toolchain() {
  REQUIRED_GO_VERSION="go1.25.1"
  GO_ARCHIVE="${REQUIRED_GO_VERSION}.linux-amd64.tar.gz"
  GO_URL="https://go.dev/dl/${GO_ARCHIVE}"

  CURRENT_GO_VERSION=""
  if command -v go >/dev/null 2>&1; then
    CURRENT_GO_VERSION="$(go version | awk '{print $3}')"
  fi

  if [ "$CURRENT_GO_VERSION" = "$REQUIRED_GO_VERSION" ]; then
    printf "Go %s already installed; skipping download.\n" "$CURRENT_GO_VERSION"
    return
  fi

  printf "Installing Go %s...\n" "$REQUIRED_GO_VERSION"

  GO_TMP_DIR=$(mktemp -d)
  GO_TARBALL="${GO_TMP_DIR}/${GO_ARCHIVE}"

  if ! curl -fsSL "$GO_URL" -o "$GO_TARBALL"; then
    rm -rf "$GO_TMP_DIR"
    die "Failed to download Go toolchain from ${GO_URL}"
  fi

  run_privileged rm -rf /usr/local/go
  run_privileged tar -C /usr/local -xzf "$GO_TARBALL"

  rm -rf "$GO_TMP_DIR"

  printf "Go %s installed to /usr/local/go.\n" "$REQUIRED_GO_VERSION"

  # Symlink go binary to /usr/local/bin for direct PATH access
  printf "Creating symlink: /usr/local/bin/go -> /usr/local/go/bin/go\n"
  run_privileged ln -sf /usr/local/go/bin/go /usr/local/bin/go
  run_privileged ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  printf "Go binaries symlinked to /usr/local/bin\n"
}

install_swag_cli() {
  if command -v swag >/dev/null 2>&1; then
    printf "swag CLI already present (%s).\n" "$(swag --version 2>/dev/null | head -n 1 || echo "version unknown")"
    return
  fi

  command -v go >/dev/null 2>&1 || die "Go command not available; cannot install swag CLI."

  printf "Installing swag CLI via go install to /usr/local/bin...\n"
  GOBIN=/usr/local/bin go install github.com/swaggo/swag/cmd/swag@latest

  if command -v swag >/dev/null 2>&1; then
    printf "swag CLI installed successfully to /usr/local/bin\n"
  else
    printf "Warning: swag CLI installation may have failed. Check /usr/local/bin/swag\n"
  fi
}

setup_openhack_user_and_group() {
  printf "Setting up openhack system user and group...\n"

  # Determine who is actually running this script (the real user, not root)
  REAL_USER="${SUDO_USER:-$(id -un)}"
  if [ "$REAL_USER" = "root" ] && [ -z "$SUDO_USER" ]; then
    printf "Running as root with no SUDO_USER. Skipping user group membership.\n"
    REAL_USER=""
  fi

  # Create openhack system user if it doesn't exist
  if id "$OPENHACK_USER" >/dev/null 2>&1; then
    printf "openhack user already exists\n"
  else
    printf "Creating openhack system user...\n"
    run_privileged useradd \
      --system \
      --home-dir "$OPENHACK_HOME" \
      --shell /usr/sbin/nologin \
      "$OPENHACK_USER"
    printf "openhack user created\n"
  fi

  # Create openhack-admins group if it doesn't exist
  if getent group "$OPENHACK_ADMIN_GROUP" >/dev/null 2>&1; then
    printf "openhack-admins group already exists\n"
  else
    printf "Creating openhack-admins group...\n"
    run_privileged groupadd "$OPENHACK_ADMIN_GROUP"
    printf "openhack-admins group created\n"
  fi

  # Add the invoking user to openhack-admins group (if not root)
  if [ -n "$REAL_USER" ] && [ "$REAL_USER" != "root" ]; then
    if id -Gn "$REAL_USER" | grep -q "$OPENHACK_ADMIN_GROUP"; then
      printf "%s is already in %s group\n" "$REAL_USER" "$OPENHACK_ADMIN_GROUP"
    else
      printf "Adding %s to %s group...\n" "$REAL_USER" "$OPENHACK_ADMIN_GROUP"
      run_privileged usermod -a -G "$OPENHACK_ADMIN_GROUP" "$REAL_USER"
      printf "%s added to %s group\n" "$REAL_USER" "$OPENHACK_ADMIN_GROUP"
      printf "Note: You may need to log out and log back in for group membership to take effect.\n"
    fi
  fi

  # Add openhack user to systemd-journal group so it can read service logs
  if id -Gn "$OPENHACK_USER" | grep -q "systemd-journal"; then
    printf "openhack user is already in systemd-journal group\n"
  else
    printf "Adding openhack user to systemd-journal group for log access...\n"
    run_privileged usermod -a -G systemd-journal "$OPENHACK_USER"
    printf "openhack user added to systemd-journal group\n"
  fi
}

setup_directories() {
  printf "Setting up hypervisor and openhack directories...\n"

  # Create all required directories
  for dir in \
    /var/hypervisor \
    /var/hypervisor/repos \
    /var/hypervisor/builds \
    /var/hypervisor/env \
    /var/hypervisor/logs \
    /var/openhack \
    /var/openhack/repos \
    /var/openhack/builds \
    /var/openhack/env \
    /var/openhack/env/template \
    /var/openhack/runtime \
    /var/openhack/runtime/logs; do

    if [ ! -d "$dir" ]; then
      printf "Creating directory %s...\n" "$dir"
      run_privileged mkdir -p "$dir"
    fi

    # Set ownership to openhack user and openhack-admins group
    run_privileged chown "$OPENHACK_USER:$OPENHACK_ADMIN_GROUP" "$dir"
    # Set permissions to 0755 (rwxr-xr-x)
    run_privileged chmod 0755 "$dir"
  done

  printf "Directory structure created and configured\n"
}

setup_sudoers() {
  printf "Configuring sudoers for openhack-admins group...\n"

  SUDOERS_FILE="/etc/sudoers.d/openhack-admins"
  SUDOERS_CONTENT="%${OPENHACK_ADMIN_GROUP} ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/bin/tee, /usr/bin/chmod, /usr/bin/chown, /usr/bin/mkdir, /usr/bin/rm, /bin/bash, /usr/bin/useradd, /usr/bin/groupadd, /usr/bin/usermod, /usr/local/bin/hyperctl"

  # Write sudoers file using printf to avoid echo issues
  printf "%s\n" "$SUDOERS_CONTENT" | run_privileged tee "$SUDOERS_FILE" >/dev/null

  # Set correct permissions on sudoers file
  run_privileged chmod 0440 "$SUDOERS_FILE"

  printf "Sudoers file configured at %s\n" "$SUDOERS_FILE"

  printf "Configuring sudoers for openhack user (service account)...\n"

  OPENHACK_SUDOERS_FILE="/etc/sudoers.d/openhack"
  OPENHACK_SUDOERS_CONTENT="${OPENHACK_USER} ALL=(ALL) NOPASSWD: /usr/bin/tee, /usr/bin/systemctl"

  # Write sudoers file for openhack user
  printf "%s\n" "$OPENHACK_SUDOERS_CONTENT" | run_privileged tee "$OPENHACK_SUDOERS_FILE" >/dev/null

  # Set correct permissions on sudoers file
  run_privileged chmod 0440 "$OPENHACK_SUDOERS_FILE"

  printf "Sudoers file configured at %s\n" "$OPENHACK_SUDOERS_FILE"
}

install_dependencies_debian() {
  printf "Installing hyperctl prerequisites for Debian-based systems (apt)...\n"

  if ! command -v apt-get >/dev/null 2>&1; then
    die "apt-get is required to install dependencies automatically. Use --nodeps to skip."
  fi

  run_privileged apt-get update
  run_privileged apt-get install -y curl redis-server nginx python3-certbot-nginx vim git

  install_go_toolchain

  if command -v go >/dev/null 2>&1; then
    printf "Go ready: %s\n" "$(go version)"
  else
    die "Go installation failed or go not on PATH."
  fi

  install_swag_cli
}

install_dependencies_fedora() {
  printf "Installing hyperctl prerequisites for Fedora/RHEL-based systems (dnf)...\n"

  if ! command -v dnf >/dev/null 2>&1; then
    die "dnf is required to install dependencies automatically. Use --nodeps to skip."
  fi

  run_privileged dnf install -y curl redis nginx certbot python3-certbot-nginx vim git

  install_go_toolchain

  if command -v go >/dev/null 2>&1; then
    printf "Go ready: %s\n" "$(go version)"
  else
    die "Go installation failed or go not on PATH."
  fi

  install_swag_cli
}

INSTALL_DEPS=1
INSTALL_DEPS_TYPE="debian"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --nodeps|-nodeps)
      INSTALL_DEPS=0
      shift
      ;;
    --fedora-deps)
      INSTALL_DEPS=1
      INSTALL_DEPS_TYPE="fedora"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

if [ "$INSTALL_DEPS" -eq 1 ]; then
  if [ "$INSTALL_DEPS_TYPE" = "fedora" ]; then
    install_dependencies_fedora
  else
    install_dependencies_debian
  fi
else
  printf "Skipping dependency installation (--nodeps).\n"
fi

# Setup openhack user, group, and directories (always, even with --nodeps)
setup_openhack_user_and_group
setup_directories
setup_sudoers

require_cmd curl
require_cmd install

OS=$(uname -s || true)
ARCH=$(uname -m || true)

[ "$OS" = "Linux" ] || die "Unsupported OS '$OS'; hyperctl currently targets Linux."
[ "$ARCH" = "x86_64" ] || [ "$ARCH" = "amd64" ] || die "Unsupported architecture '$ARCH'; hyperctl currently targets x86_64."

TMP_DIR=$(mktemp -d)
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT INT TERM

TMP_BIN="$TMP_DIR/$BINARY_NAME"
printf "Downloading %s to temporary folder %s...\n" "$ASSET_NAME" "$TMP_BIN"

curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN" || die "Failed to download asset from $DOWNLOAD_URL"
chmod 0755 "$TMP_BIN"

TARGET="$INSTALL_DIR/$BINARY_NAME"

install_binary() {
  install -m 0755 "$TMP_BIN" "$TARGET"
}

if [ -w "$INSTALL_DIR" ]; then
  install_binary
else
  if command -v sudo >/dev/null 2>&1; then
    printf "Elevating privileges to write into %s...\n" "$INSTALL_DIR"
    sudo install -m 0755 "$TMP_BIN" "$TARGET"
  else
    die "Cannot write to $INSTALL_DIR. Re-run with sudo or set INSTALL_DIR to a writable path."
  fi
fi

printf "hyperctl installed to %s\n" "$TARGET"

if command -v "$BINARY_NAME" >/dev/null 2>&1; then
  printf "Detected %s version: %s\n" "$BINARY_NAME" "$($BINARY_NAME version 2>/dev/null || echo 'unknown')"
else
  err "Binary not found on PATH; ensure ${INSTALL_DIR} is in your PATH."
fi
