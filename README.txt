Shutdown Timer
==============

Choose Shut down, Restart, Sleep, Hibernate or Log out. Click a preset to
start immediately, or enter a custom delay / clock time and click Start timer.

INSTALL / UPDATE
----------------
In this extracted folder, run:
    bash install.sh

Open Shutdown Timer from your application menu. Quit an older running copy
before reopening after an update. No sudo or compiler is needed.
To run without installing: ./shutdown-timer

CONTROLS
--------
Pause / Resume freezes or resumes the countdown.
+5 min / +15 min adds time, even while paused.
Cancel timer clears it. A new preset or custom timer replaces the old one.
Cancel first if you want to change the action.
Notify before selects an advance desktop notification, or Off.
Hide to tray keeps the timer running, with controls in its tray menu.
Quitting cancels the timer. Running timers never restart at login.

CUSTOM INPUT
------------
45        = 45 minutes from now
90m       = 90 minutes from now
1h30m     = 1 hour 30 minutes from now (1h 30m also works)
45s       = 45 seconds from now
23:30     = next local 23:30, today or tomorrow

SAFE TRIAL
----------
    ./shutdown-timer --demo

Try 5s. Demo mode never sends a power request.

KDE / POWER ACTIONS
-------------------
Shutdown, restart and logout use KDE's session manager when running in KDE.
KDE's session settings and apps' unsaved-work prompts are respected. Apps
can hold up or cancel shutdown. The separate KDE confirmation is skipped.
This corrects the previous direct system shutdown which could leave KDE
restoring an older saved session. Manually saved sessions and autostart are
still controlled by KDE / the apps; no session files are edited.

Sleep and hibernate need system support. Logout requires KDE.
Usually no password is needed in an active desktop session. Errors appear
in the app. Check desktop permissions / support if an action fails.

The timer catches up after sleep but cannot wake the computer. It only runs
while the app is open, including when hidden in the tray.

UNINSTALL
---------
Delete:
    ~/.local/bin/shutdown-timer
    ~/.local/share/applications/shutdown-timer.desktop
    ~/.local/share/icons/shutdown-timer.png
    ~/.local/share/icons/hicolor/512x512/apps/shutdown-timer.png

If you installed with XDG_DATA_HOME / XDG_BIN_HOME overrides, use those
locations instead. See the repository README for building and testing.
