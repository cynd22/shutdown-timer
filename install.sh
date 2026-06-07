#!/usr/bin/env bash
# One-step installer for Shutdown Timer.
# Copies the app into your user folders and adds it to the application menu.
# No sudo needed — everything goes under your home directory.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bin_dir="$HOME/.local/bin"
app_dir="$HOME/.local/share/applications"
icon_dir="$HOME/.local/share/icons"
mkdir -p "$bin_dir" "$app_dir" "$icon_dir"

install -m 755 "$here/shutdown-timer" "$bin_dir/shutdown-timer"
[ -f "$here/icon.png" ] && install -m 644 "$here/icon.png" "$icon_dir/shutdown-timer.png"

cat > "$app_dir/shutdown-timer.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Shutdown Timer
Comment=Power off after a delay or at a set time
Exec=$bin_dir/shutdown-timer
Icon=$icon_dir/shutdown-timer.png
Terminal=false
Categories=Utility;System;
EOF
chmod 644 "$app_dir/shutdown-timer.desktop"

# Refresh the menu cache if the tool is available (harmless if it isn't).
command -v update-desktop-database >/dev/null 2>&1 && \
  update-desktop-database "$app_dir" >/dev/null 2>&1 || true

echo "Done!  'Shutdown Timer' is now in your application menu."
echo "(Search for it the way you'd open any other app.)"
echo "If it doesn't show up immediately, log out and back in once."
