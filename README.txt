Shutdown Timer
==============

A little app that powers off the computer after a delay (15 min, 1 hour, a
custom number of minutes) or at a set clock time (e.g. 23:30). It shows a
live countdown and has a Cancel button.

It does NOT need an internet connection and does NOT need to be "installed"
in the system sense — everything stays in your home folder.


HOW TO USE IT (the easy way)
----------------------------
1. Open a terminal in this folder.
   (In your file manager: right-click empty space -> "Open Terminal Here",
    or open Terminal and type:  cd  then drag this folder onto the window,
    then press Enter.)

2. Run:
       bash install.sh

3. That's it. Open your application menu / app launcher and search for
   "Shutdown Timer". Click it like any other app.


JUST WANT TO RUN IT ONCE WITHOUT INSTALLING?
--------------------------------------------
In a terminal in this folder:
       ./shutdown-timer
(If it says permission denied, first run:  chmod +x shutdown-timer )


DOES IT NEED A PASSWORD / SUDO?
-------------------------------
Usually no. On a normal desktop login, your user is allowed to power off
without a password, so the app just works. If your system is locked down
and powering off fails, the app will show an error explaining you'd need
to run it from a terminal with sudo.


WHAT YOU TYPE IN THE CUSTOM BOX
-------------------------------
   45        -> 45 minutes from now
   90m       -> 90 minutes from now
   1h30m     -> 1 hour 30 minutes from now
   45s       -> 45 seconds from now
   23:30     -> at 23:30 today (or tomorrow if it's already past)


TO UNINSTALL
------------
Delete these (safe to remove by hand):
   ~/.local/bin/shutdown-timer
   ~/.local/share/applications/shutdown-timer.desktop
   ~/.local/share/icons/shutdown-timer.png
