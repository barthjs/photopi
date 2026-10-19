#!/usr/bin/env bash

set -e -o pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

REPO="barthjs/photopi"
APP_NAME="photopi"
# APP_TITLE and APP_COMMENT name the app in the services and the launchers
APP_TITLE="PhotoPi"
APP_COMMENT="Turns a computer with a camera and a screen into a photo booth"
INSTALL_MODE="" # "user" or "system"

print_info() {
  echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
  echo -e "${GREEN}✓${NC} $1"
}

print_warn() {
  echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
  echo -e "${RED}✗${NC} $1"
}

show_help() {
  cat << 'EOF'
Usage: install.sh [OPTION]

Options:
  --user      Install for current user only (systemd user services)
  --system    Install system-wide (requires sudo, runs as current user, without kiosk browser)
  --help      Show this help message

If no option is provided, you will be prompted to choose.
EOF
}

# parse_args reads --user/--system/--help from "$@" into INSTALL_MODE.
parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
      --user)
        INSTALL_MODE="user"
        shift
        ;;
      --system)
        INSTALL_MODE="system"
        shift
        ;;
      --help)
        show_help
        exit 0
        ;;
      *)
        print_error "Unknown option: $1"
        show_help
        exit 1
        ;;
    esac
  done
}

# choose_install_mode prompts for INSTALL_MODE if it wasn't set by a flag.
choose_install_mode() {
  if [ -n "$INSTALL_MODE" ]; then
    return
  fi

  echo "Choose installation mode:"
  echo "  1) User-wide   (installs for $USER only, uses systemd user services)"
  echo "  2) System-wide (installs for all users, requires sudo, runs as $USER, without kiosk browser)"
  read -p "Select mode (1 or 2): " -n 1 -r
  echo
  case $REPLY in
    1)
      INSTALL_MODE="user"
      ;;
    2)
      INSTALL_MODE="system"
      ;;
    *)
      print_error "Invalid choice"
      exit 1
      ;;
  esac
}

# detect_arch maps "uname -m" to the arch suffix used in release archive names.
detect_arch() {
  local machine
  machine=$(uname -m)
  case $machine in
    x86_64)
      echo "amd64"
      ;;
    i686 | i386)
      echo "386"
      ;;
    armv6l)
      echo "armv6"
      ;;
    armv7l)
      echo "armv7"
      ;;
    aarch64)
      echo "arm64"
      ;;
    riscv64)
      echo "riscv64"
      ;;
    *)
      print_error "Unsupported architecture: $machine"
      exit 1
      ;;
  esac
}

# get_asset_url looks up a release asset's API url by name
# (or, with -E, by extended-regex name pattern) in $RELEASE_JSON.
get_asset_url() {
  grep "${2:--F}" -B 5 "\"name\": \"$1\"" <<< "$RELEASE_JSON" | grep '"url":' | grep '/assets/' | head -1 | cut -d '"' -f 4 || true
}

# download_release fetches $BINARY_NAME (and its checksums, if published)
# from the latest GitHub release into $TMP_DIR, verifying the checksum when available.
download_release() {
  local curl_auth=()
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl_auth=(-H "Authorization: token $GITHUB_TOKEN")
  fi

  print_info "Downloading latest ${APP_NAME} binary for Linux ${ARCH}..."

  if ! RELEASE_JSON=$(curl -fsS "${curl_auth[@]}" "https://api.github.com/repos/${REPO}/releases/latest"); then
    print_error "Failed to fetch release information from GitHub"
    exit 1
  fi

  local release_url
  release_url=$(get_asset_url "$BINARY_NAME")
  if [ -z "$release_url" ]; then
    print_error "Failed to find binary for $BINARY_NAME"
    exit 1
  fi

  print_info "Downloading from: $release_url"

  if ! curl -fsL "${curl_auth[@]}" -H "Accept: application/octet-stream" "$release_url" -o "$TMP_DIR/$BINARY_NAME"; then
    print_error "Failed to download binary"
    exit 1
  fi

  local checksums_url
  checksums_url=$(get_asset_url '.*checksums.*' -E)
  if [ -n "$checksums_url" ]; then
    curl -fsL "${curl_auth[@]}" -H "Accept: application/octet-stream" "$checksums_url" -o "$TMP_DIR/checksums.txt" 2>/dev/null || true
  fi

  verify_checksum
}

# verify_checksum checks $TMP_DIR/$BINARY_NAME against $TMP_DIR/checksums.txt
# when both a checksums file was published and sha256sum is available locally.
verify_checksum() {
  if [ ! -f "$TMP_DIR/checksums.txt" ]; then
    print_warn "Checksums file not available, skipping integrity check"
    return
  fi
  if ! command -v sha256sum &> /dev/null; then
    print_warn "sha256sum not found, skipping integrity check"
    return
  fi

  if grep "$BINARY_NAME" "$TMP_DIR/checksums.txt" | sed 's/^\.\/*//' | (cd "$TMP_DIR" && sha256sum -c - > /dev/null 2>&1); then
    print_success "Binary integrity verified"
  else
    print_error "Binary integrity check failed"
    exit 1
  fi
}

# as_install_user runs "$@" as the user, via sudo when INSTALL_MODE is "system", directly otherwise.
as_install_user() {
  if [ "$INSTALL_MODE" = "system" ]; then
    sudo "$@"
  else
    "$@"
  fi
}

# write_file writes content to path, via sudo tee when INSTALL_MODE is "system", directly otherwise.
write_file() {
  local path="$1"
  local content="$2"
  if [ "$INSTALL_MODE" = "system" ]; then
    echo "$content" | sudo tee "$path" > /dev/null
  else
    echo "$content" > "$path"
  fi
}

# systemctl_user runs "systemctl" against the right instance for INSTALL_MODE:
# the current user's --user instance, or the system instance via sudo.
systemctl_user() {
  if [ "$INSTALL_MODE" = "system" ]; then
    sudo systemctl "$@"
  else
    systemctl --user "$@"
  fi
}

# install_binary downloads, verifies, and installs the release binary.
install_binary() {
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"' EXIT

  download_release

  cd "$TMP_DIR"
  if ! tar -xzf "$BINARY_NAME" 2>/dev/null; then
    print_error "Failed to extract binary from archive"
    exit 1
  fi

  local extracted="$TMP_DIR/$APP_NAME"
  if [ ! -f "$extracted" ]; then
    print_error "Failed to extract binary from archive"
    exit 1
  fi
  chmod +x "$extracted"

  BINARY_PATH="$INSTALL_DIR/$APP_NAME"
  as_install_user mv "$extracted" "$BINARY_PATH"
  if [ "$INSTALL_MODE" = "system" ]; then
    sudo chown root:root "$BINARY_PATH"
  fi
  print_success "Binary installed to $BINARY_PATH"

  if [ "$INSTALL_MODE" = "user" ] && [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    print_warn "$INSTALL_DIR is not in your PATH. Add it with:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
  fi
}

# write_desktop_entry writes a .desktop launcher named file to $APPS_DIR.
write_desktop_entry() {
  local file="$1"
  local name="$2"
  local comment="$3"
  local exec_line="$4"

  read -r -d '' desktop_content << EOF || true
[Desktop Entry]
Name=$name
Comment=$comment
Type=Application
Exec=$exec_line
Terminal=false
Icon=camera-photo
Categories=Utility;
EOF

  write_file "$APPS_DIR/$file" "$desktop_content"
  print_success "Desktop shortcut created at $APPS_DIR/$file"
}

# install_desktop_entries writes the launchers of the app. The launcher $APP_TITLE always exists:
# without a service it runs the binary, which opens the browser itself, with the service it starts the service
# and opens the interface in the browser, and with the kiosk it starts both services.
# With the kiosk, two more launchers start only the application or only the kiosk.
install_desktop_entries() {
  if [ "$INSTALL_MODE" = "system" ]; then
    # A system service runs without a desktop session, so there is nothing to launch
    return
  fi

  if [ "$APP_SERVICE" != "yes" ]; then
    write_desktop_entry "$APP_NAME.desktop" "$APP_TITLE" "$APP_COMMENT" "$BINARY_PATH"
    return
  fi

  if [ "$KIOSK_SERVICE" != "yes" ]; then
    # The service opens no browser, so the launcher opens the interface once the service had time to start
    write_desktop_entry "$APP_NAME.desktop" "$APP_TITLE" "$APP_COMMENT" "sh -c \"systemctl --user start $APP_NAME.service && sleep 2 && xdg-open http://localhost:8080\""
    return
  fi

  write_desktop_entry "$APP_NAME.desktop" "$APP_TITLE" "$APP_TITLE with the kiosk browser" "systemctl --user start $APP_NAME.service $APP_NAME-kiosk.service"
  write_desktop_entry "$APP_NAME-server.desktop" "$APP_TITLE Server" "$APP_TITLE without the kiosk browser" "systemctl --user start $APP_NAME.service"
  write_desktop_entry "$APP_NAME-kiosk.desktop" "$APP_TITLE Kiosk" "$APP_TITLE kiosk browser" "systemctl --user start $APP_NAME-kiosk.service"
}

# install_service writes content as a systemd unit for name to $SYSTEMD_DIR,
# reloads the systemd daemon, and offers to enable and start it.
install_service() {
  local name="$1"
  local content="$2"
  local unit_path="$SYSTEMD_DIR/$name"

  write_file "$unit_path" "$content"
  print_success "Service file created at $unit_path"

  systemctl_user daemon-reload

  echo ""
  read -p "Enable and start $name now? (Y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]] || [ -z "$REPLY" ]; then
    systemctl_user enable "$name"
    systemctl_user start "$name"
    print_success "$name enabled and started"
    if [ "$INSTALL_MODE" = "system" ]; then
      echo "Check status with: sudo systemctl status $name"
    else
      echo "Check status with: systemctl --user status $name"
      print_info "User services start with every login of $USER, so an automatic login starts them on boot"
    fi
  fi
}

# install_app_service offers to install the application itself as a systemd service.
# The user service is the one of the Debian package with the path of the binary.
# The system service runs as $USER on a machine without a desktop session.
install_app_service() {
  APP_SERVICE="no"

  echo ""
  read -p "Do you want to install $APP_NAME as a systemd service? (Y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]] && [ -n "$REPLY" ]; then
    print_info "Skipped service installation"
    echo ""
    echo "You can run $APP_NAME directly by:"
    echo "  1. Starting it from your applications launcher"
    echo "  2. Running: $APP_NAME"
    return
  fi

  local service_content
  if [ "$INSTALL_MODE" = "user" ]; then
    read -r -d '' service_content << EOF || true
[Unit]
Description=$APP_TITLE

[Service]
Type=simple
Environment=PHOTOPI_SUPERVISED=1
ExecStart=$BINARY_PATH
Restart=always
RestartSec=5
# Only the application itself is stopped, so a system command it started,
# such as restarting both services from the admin panel, still finishes once it stopped this service.
KillMode=process

[Install]
WantedBy=default.target
EOF
  else
    read -r -d '' service_content << EOF || true
[Unit]
Description=$APP_TITLE
After=network.target

[Service]
Type=simple
User=$USER
SupplementaryGroups=video plugdev
# There are no user services to control, so the system actions of the admin panel stay disabled.
# Exiting the process from the admin panel restarts it through Restart=.
Environment=SYSTEM_SERVER_RESTART_COMMAND= SYSTEM_SERVER_STOP_COMMAND= SYSTEM_KIOSK_RESTART_COMMAND= SYSTEM_KIOSK_STOP_COMMAND= SYSTEM_ALL_RESTART_COMMAND= SYSTEM_ALL_STOP_COMMAND=
LogsDirectory=$APP_NAME
WorkingDirectory=$DATA_DIR
Environment=PHOTOPI_SUPERVISED=1
ExecStart=$BINARY_PATH
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  fi

  print_info "Installing as a systemd $INSTALL_MODE service..."
  install_service "$APP_NAME.service" "$service_content"
  APP_SERVICE="yes"
}

# install_kiosk_service offers to install the Chromium kiosk browser as a user service, the one of the Debian package.
# It runs in the graphical session of the user, which a system service has no access to,
# so a system installation offers no kiosk.
install_kiosk_service() {
  KIOSK_SERVICE="no"

  if [ "$INSTALL_MODE" != "user" ] || [ "$APP_SERVICE" != "yes" ]; then
    return
  fi

  echo ""
  read -p "Do you also want to install a Chromium kiosk service that opens the app full-screen? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    return
  fi

  if ! command -v chromium &> /dev/null && ! command -v chromium-browser &> /dev/null; then
    print_warn "Chromium was not found on this system. Install it first, then re-run this step."
    return
  fi
  if ! command -v curl &> /dev/null; then
    print_warn "curl was not found, so the kiosk waits a fixed time for the app instead of until it answers."
  fi

  local kiosk_content
  read -r -d '' kiosk_content << EOF || true
[Unit]
Description=$APP_TITLE kiosk browser
# Starting the kiosk starts the application, and the kiosk always starts after it
Wants=$APP_NAME.service
After=$APP_NAME.service

[Service]
Type=simple
# Chromium does not reload its error page by itself, so it only starts once the application answers.
# A failed wait ends the start, and Restart= tries again.
ExecStartPre=/bin/sh -c 'command -v curl >/dev/null || { sleep 5; exit 0; }; for i in \$\$(seq 60); do curl -fsS -o /dev/null http://localhost:8080/healthz && exit 0; sleep 1; done; exit 1'
ExecStart=/bin/sh -c 'exec "\$(command -v chromium || command -v chromium-browser)" --kiosk --app=http://localhost:8080 --password-store=basic --no-first-run --noerrdialogs --disable-infobars --disable-session-crashed-bubble --disable-pinch --overscroll-history-navigation=0 --force-renderer-accessibility'
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
EOF

  install_service "$APP_NAME-kiosk.service" "$kiosk_content"
  KIOSK_SERVICE="yes"

  print_info "It points at http://localhost:8080, edit the unit if the app runs on a different port."
  print_info "--force-renderer-accessibility lets an on-screen keyboard (e.g. matchbox-keyboard, onboard) detect focused text fields; install and start one yourself if you need one on a touchscreen without a physical keyboard attached."
}

print_next_steps() {
  echo ""
  print_success "Installation complete!"

  if [ "$INSTALL_MODE" = "user" ] && [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Start the app with: $BINARY_PATH"
  else
    echo "Start the app with: $APP_NAME"
  fi
}

main() {
  parse_args "$@"

  echo -e "${GREEN}=== ${APP_NAME} installer ===${NC}\n"

  ARCH=$(detect_arch)
  BINARY_NAME="${APP_NAME}_linux_${ARCH}.tar.gz"

  choose_install_mode
  print_info "Installation mode: $INSTALL_MODE"

  if [ "$INSTALL_MODE" = "user" ]; then
    APPS_DIR="${HOME}/.local/share/applications"
    CONFIG_DIR="${HOME}/.config/${APP_NAME}"
    DATA_DIR="${HOME}/.local/share/${APP_NAME}"
    INSTALL_DIR="${HOME}/.local/bin"
    SYSTEMD_DIR="${HOME}/.config/systemd/user"
  else
    APPS_DIR="/usr/share/applications"
    CONFIG_DIR="/etc/${APP_NAME}"
    DATA_DIR="/var/lib/${APP_NAME}"
    INSTALL_DIR="/usr/local/bin"
    SYSTEMD_DIR="/etc/systemd/system"
  fi

  as_install_user mkdir -p "$APPS_DIR" "$CONFIG_DIR" "$DATA_DIR" "$INSTALL_DIR" "$SYSTEMD_DIR"
  if [ "$INSTALL_MODE" = "system" ]; then
    sudo chown "$USER:$USER" "$CONFIG_DIR" "$DATA_DIR"
  fi

  install_binary
  install_app_service
  install_kiosk_service
  install_desktop_entries
  print_next_steps
}

main "$@"
