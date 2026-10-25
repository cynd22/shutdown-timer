#!/usr/bin/env bash
# One-step installer for Shutdown Timer.
# Copies the app into your user folders and adds it to the application menu.
# No sudo needed — everything goes under your home directory.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bin_dir="${XDG_BIN_HOME:-$HOME/.local/bin}"
data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
app_dir="$data_dir/applications"
icon_dir="$data_dir/icons"
theme_icon_dir="$icon_dir/hicolor/512x512/apps"

if [[ ! -f "$here/shutdown-timer" ]]; then
  echo "No shutdown-timer executable found beside install.sh." >&2
  echo "Extract the release archive, or build the project first." >&2
  exit 1
fi
mkdir -p "$bin_dir" "$app_dir" "$theme_icon_dir"

install -m 755 "$here/shutdown-timer" "$bin_dir/shutdown-timer"
if [[ -f "$here/icon.png" ]]; then
  install -m 644 "$here/icon.png" "$icon_dir/shutdown-timer.png"
  install -m 644 "$here/icon.png" "$theme_icon_dir/shutdown-timer.png"
fi

# Quote the executable as one argument, then escape the Desktop Entry string.
exec_path="$bin_dir/shutdown-timer"
exec_path="${exec_path//\\/\\\\}"
exec_path="${exec_path//\"/\\\"}"
exec_path="${exec_path//\$/\\\$}"
exec_path="${exec_path//\`/\\\`}"
exec_path="${exec_path//%/%%}"
exec_path="${exec_path//\\/\\\\}"

cat > "$app_dir/shutdown-timer.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Shutdown Timer
Comment=Schedule shutdown, restart, sleep, hibernate or logout
Exec=/usr/bin/env "$exec_path"
Icon=shutdown-timer
Terminal=false
Categories=Utility;
StartupNotify=true
EOF
chmod 644 "$app_dir/shutdown-timer.desktop"

# Refresh the menu cache if the tool is available (harmless if it isn't).
command -v update-desktop-database >/dev/null 2>&1 && \
  update-desktop-database "$app_dir" >/dev/null 2>&1 || true
if command -v kbuildsycoca6 >/dev/null 2>&1; then
  kbuildsycoca6 --noincremental >/dev/null 2>&1 || true
elif command -v kbuildsycoca5 >/dev/null 2>&1; then
  kbuildsycoca5 --noincremental >/dev/null 2>&1 || true
fi

echo "Done!  'Shutdown Timer' is now in your application menu."
echo "(Search for it the way you'd open any other app.)"
echo "If updating, quit any older copy before opening the new one."
