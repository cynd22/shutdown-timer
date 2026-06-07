# Shutdown Timer

A tiny GUI to power off a Linux machine after a delay (15 min, 1 hour, a
custom number of minutes) or at a set clock time (e.g. `23:30`). It shows a
live countdown and has a Cancel button.

It ships as a **single self-contained binary** — no Python, no runtime, no
internet connection required. On a normal desktop login it needs **no sudo**,
because logind/polkit already lets the active user power off.

<img src="https://raw.githubusercontent.com/cynd22/shutdown-timer/v1.0.2/icon.png" alt="Shutdown Timer" width="96">

## Install (easiest)

1. Download `ShutdownTimer.tar.gz` from the
   [latest release](../../releases/latest) and extract it.
2. Open a terminal in the extracted folder and run:
   ```bash
   bash install.sh
   ```
3. Open your application menu and search for **Shutdown Timer**. Done.

Everything installs under your home folder (`~/.local/...`) — no system changes.

## Run once without installing

```bash
./shutdown-timer        # if needed first: chmod +x shutdown-timer
```

## Custom delay / time box

| You type | Means |
|----------|-------|
| `45`     | 45 minutes from now |
| `90m`    | 90 minutes from now |
| `1h30m`  | 1 hour 30 minutes from now |
| `45s`    | 45 seconds from now |
| `23:30`  | at 23:30 today (or tomorrow if already past) |

## Does it need a password?

Usually no. If your system is locked down and powering off fails, the app
shows an error explaining you'd need to run it from a terminal with `sudo`.

## Build from source

Requires Go 1.21+, a C compiler, and the X11/OpenGL dev headers Fyne needs.

```bash
go build -ldflags "-s -w" -o shutdown-timer .
```

## Uninstall

```bash
rm -f ~/.local/bin/shutdown-timer \
      ~/.local/share/applications/shutdown-timer.desktop \
      ~/.local/share/icons/shutdown-timer.png
```

## License

MIT
