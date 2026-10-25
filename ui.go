package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/godbus/dbus/v5"
)

type timerUI struct {
	app     fyne.App
	window  fyne.Window
	demo    bool
	execute func(context.Context, powerAction) error
	timer   timerState
	busy    bool
	message string

	action    *widget.Select
	custom    *widget.Entry
	warning   *widget.Select
	start     *widget.Button
	pause     *widget.Button
	cancel    *widget.Button
	extra5    *widget.Button
	extra15   *widget.Button
	presets   []*widget.Button
	countdown *canvas.Text
	status    *widget.Label
	progress  *widget.ProgressBar
	tray      desktop.App
	trayMenu  *fyne.Menu
	trayState *fyne.MenuItem
	trayPause *fyne.MenuItem
	trayExtra *fyne.MenuItem
	trayStop  *fyne.MenuItem
}

func newTimerUI(a fyne.App, w fyne.Window, demo bool, execute func(context.Context, powerAction) error) *timerUI {
	u := &timerUI{app: a, window: w, demo: demo, execute: execute, message: "Nothing scheduled"}
	prefs := a.Preferences()
	u.action = widget.NewSelect(actionNames, func(value string) { prefs.SetString("action", value) })
	selected := prefs.StringWithFallback("action", string(shutdown))
	if !validAction(powerAction(selected)) {
		selected = string(shutdown)
	}
	u.action.SetSelected(selected)
	u.custom = widget.NewEntry()
	u.custom.SetPlaceHolder("45, 1h30m, or 23:30")
	u.custom.SetText(prefs.StringWithFallback("custom", ""))
	u.start = widget.NewButtonWithIcon("Start timer", theme.MediaPlayIcon(), u.startCustom)
	u.start.Importance = widget.HighImportance
	u.custom.OnSubmitted = func(string) { u.startCustom() }

	u.warning = widget.NewSelect([]string{"Off", "30 seconds", "1 minute", "5 minutes"}, func(value string) { prefs.SetString("warning", value) })
	warning := prefs.StringWithFallback("warning", "1 minute")
	switch warning {
	case "Off", "30 seconds", "1 minute", "5 minutes":
	default:
		warning = "1 minute"
	}
	u.warning.SetSelected(warning)

	u.countdown = canvas.NewText("--:--:--", theme.ForegroundColor())
	u.countdown.Alignment = fyne.TextAlignCenter
	u.countdown.TextSize = 44
	u.countdown.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	u.status = widget.NewLabel(u.message)
	u.status.Alignment = fyne.TextAlignCenter
	u.status.Wrapping = fyne.TextWrapWord
	u.progress = widget.NewProgressBar()
	u.progress.TextFormatter = func() string { return fmt.Sprintf("%d%% elapsed", int(u.progress.Value*100)) }
	u.pause = widget.NewButtonWithIcon("Pause", theme.MediaPauseIcon(), func() { u.togglePause() })
	u.cancel = widget.NewButtonWithIcon("Cancel timer", theme.CancelIcon(), u.cancelTimer)
	u.extra5 = widget.NewButton("+5 min", func() { u.extend(5 * time.Minute) })
	u.extra15 = widget.NewButton("+15 min", func() { u.extend(15 * time.Minute) })

	presetGrid := container.NewGridWithColumns(4)
	for _, preset := range []struct {
		label string
		delay time.Duration
	}{{"15 min", 15 * time.Minute}, {"30 min", 30 * time.Minute}, {"1 hour", time.Hour}, {"2 hours", 2 * time.Hour}} {
		button := widget.NewButton(preset.label, func() { u.schedule(preset.delay) })
		u.presets = append(u.presets, button)
		presetGrid.Add(button)
	}
	heading := widget.NewLabelWithStyle("Shutdown Timer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("Choose an action and a time.")
	icon := canvas.NewImageFromResource(a.Icon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(48, 48))
	header := container.NewBorder(nil, nil, icon, nil, container.NewVBox(heading, subtitle))
	if demo {
		header = container.NewVBox(header, widget.NewLabelWithStyle("Demo mode · power actions are disabled", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	}
	card := widget.NewCard("", "", container.NewVBox(
		container.NewPadded(u.countdown), u.status, u.progress,
		container.NewGridWithColumns(3, u.pause, u.extra5, u.extra15),
		u.cancel,
	))
	helper := widget.NewLabel("Minutes, a duration, or a 24-hour clock time.\nA clock time that has passed means tomorrow.")
	helper.Wrapping = fyne.TextWrapWord
	help := widget.NewButtonWithIcon("How it works", theme.InfoIcon(), func() {
		dialog.ShowInformation("How it works",
			"Preset buttons start immediately. Starting another timer replaces the current one.\n\n"+
				"Pause freezes the countdown. +5 min and +15 min postpone it.\n\n"+
				"On KDE, shutdown, restart and logout use the session manager so your session settings and unsaved-work prompts are respected. Apps can delay or cancel the action.\n\n"+
				"Sleep and hibernate depend on your system's support. Logout requires KDE.\n\n"+
				"The timer runs while this app is open, including when hidden in the tray. It catches up after sleep; it cannot wake a sleeping computer. Quitting cancels it. Timers are never restored on login.\n\n"+
				"For a harmless trial, run shutdown-timer --demo.\nVersion: "+version, w)
	})
	helperRow := container.NewBorder(nil, nil, widget.NewLabel("Notify before"), nil, u.warning)
	content := container.NewVBox(
		header, card,
		container.NewBorder(nil, nil, widget.NewLabel("When time is up"), nil, u.action),
		widget.NewLabelWithStyle("Start a quick timer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		presetGrid,
		widget.NewLabel("Or choose a delay / time"),
		container.NewBorder(nil, nil, nil, u.start, u.custom),
		helper, helperRow,
	)
	footer := container.NewHBox(help, layout.NewSpacer())
	if tray, ok := a.(desktop.App); ok {
		u.tray = tray
		footer.Add(widget.NewButtonWithIcon("Hide to tray", theme.VisibilityOffIcon(), u.hideToTray))
	}
	w.SetContent(container.NewPadded(container.NewBorder(nil, footer, nil, nil, container.NewVScroll(content))))
	u.refresh(time.Now())
	return u
}

func (u *timerUI) startCustom() {
	if u.busy {
		return
	}
	delay, err := parseWhenAt(u.custom.Text, time.Now())
	if err != nil {
		dialog.ShowError(err, u.window)
		return
	}
	u.app.Preferences().SetString("custom", strings.TrimSpace(u.custom.Text))
	u.schedule(delay)
}

func (u *timerUI) schedule(delay time.Duration) {
	if u.busy || delay <= 0 || delay > maxDelay {
		return
	}
	now := time.Now()
	u.timer.start(now, delay, powerAction(u.action.Selected))
	u.message = ""
	u.refresh(now)
}

func (u *timerUI) togglePause() {
	u.timer.togglePause(time.Now())
	u.refresh(time.Now())
}

func (u *timerUI) extend(delay time.Duration) {
	u.timer.extend(time.Now(), delay)
	u.refresh(time.Now())
}

func (u *timerUI) cancelTimer() {
	if !u.timer.active {
		return
	}
	u.timer = timerState{}
	u.message = "Cancelled · nothing scheduled"
	u.refresh(time.Now())
}

func (u *timerUI) warningDelay() time.Duration {
	switch u.warning.Selected {
	case "30 seconds":
		return 30 * time.Second
	case "1 minute":
		return time.Minute
	case "5 minutes":
		return 5 * time.Minute
	}
	return 0
}

func (u *timerUI) tick(now time.Time) {
	warn, fire := u.timer.advance(now, u.warningDelay())
	if warn {
		u.notify("Shutdown Timer", fmt.Sprintf("%s in %s. Open the timer to pause, add time, or cancel.", u.timer.action, fmtDur(u.timer.remaining(now))))
	}
	if fire {
		u.fire()
	}
	u.refresh(now)
}

func (u *timerUI) fire() {
	action := u.timer.action
	if u.demo {
		u.message = "Demo complete · " + strings.ToLower(string(action)) + " was not run"
		u.notify("Shutdown Timer · demo complete", "No power action was run.")
		return
	}
	u.busy = true
	u.message = "Requesting " + strings.ToLower(string(action)) + "…"
	go func() {
		err := u.execute(context.Background(), action)
		fyne.Do(func() {
			u.busy = false
			if err != nil {
				u.message = string(action) + " failed · nothing scheduled"
				u.window.Show()
				u.window.RequestFocus()
				u.notify("Shutdown Timer", string(action)+" failed. Open the app for details.")
				dialog.ShowError(err, u.window)
			} else {
				u.message = string(action) + " requested · nothing scheduled"
			}
			u.refresh(time.Now())
		})
	}()
}

func (u *timerUI) refresh(now time.Time) {
	text := u.message
	title := "Shutdown Timer"
	digits := "--:--:--"
	progress := 0.0
	if u.timer.active {
		left := u.timer.remaining(now)
		digits = fmtDur(left)
		progress = max(0, min(1, 1-float64(left)/float64(u.timer.total)))
		text = string(u.timer.action) + " · " + u.timer.deadline.Format("Mon 2 Jan, 15:04:05")
		if u.timer.paused {
			text = "Paused · " + strings.ToLower(string(u.timer.action))
		}
		title = digits + " · " + string(u.timer.action)
		if u.timer.paused {
			title = "Paused · " + title
		}
	} else if u.timer.total > 0 {
		digits, progress = "00:00:00", 1
	}
	if u.demo {
		title += " · Demo"
	}
	if u.countdown.Text != digits {
		u.countdown.Text = digits
		u.countdown.Refresh()
	}
	// Refresh the color after a desktop theme change too.
	if u.countdown.Color != theme.ForegroundColor() {
		u.countdown.Color = theme.ForegroundColor()
		u.countdown.Refresh()
	}
	if u.status.Text != text {
		u.status.SetText(text)
	}
	if u.window.Title() != title {
		u.window.SetTitle(title)
	}
	if u.progress.Value != progress {
		u.progress.SetValue(progress)
	}
	for _, button := range []*widget.Button{u.pause, u.cancel, u.extra5, u.extra15} {
		setEnabled(button, u.timer.active)
	}
	pauseLabel := "Pause"
	pauseIcon := theme.MediaPauseIcon()
	if u.timer.paused {
		pauseLabel, pauseIcon = "Resume", theme.MediaPlayIcon()
	}
	if u.pause.Text != pauseLabel {
		u.pause.SetText(pauseLabel)
		u.pause.SetIcon(pauseIcon)
	}
	// Lock action selection to the action actually armed. Presets/custom can
	// still replace it; changing action first requires cancelling the timer.
	setEnabled(u.action, !u.timer.active && !u.busy)
	setEnabled(u.start, !u.busy)
	setEnabled(u.custom, !u.busy)
	for _, button := range u.presets {
		setEnabled(button, !u.busy)
	}
	u.refreshTray(text, pauseLabel)
}

func setEnabled(widget fyne.Disableable, enabled bool) {
	if widget.Disabled() == enabled {
		if enabled {
			widget.Enable()
		} else {
			widget.Disable()
		}
	}
}

func (u *timerUI) notify(title, body string) {
	// An unavailable notification server must never stall the countdown.
	go u.app.SendNotification(fyne.NewNotification(title, body))
}

func (u *timerUI) hideToTray() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		available := false
		if conn, err := dbus.ConnectSessionBus(); err == nil {
			defer conn.Close()
			_ = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.NameHasOwner", 0, "org.kde.StatusNotifierWatcher").Store(&available)
		}
		fyne.Do(func() {
			if available {
				u.window.Hide()
			} else {
				dialog.ShowInformation("System tray unavailable", "Your desktop's system tray is unavailable. The timer will keep running in this window.", u.window)
			}
		})
	}()
}

func (u *timerUI) requestQuit() {
	if u.busy {
		dialog.ShowInformation("Action in progress", "The request is already being sent to your desktop. Wait for it to finish.", u.window)
		return
	}
	quit := func() {
		u.timer = timerState{}
		u.window.SetCloseIntercept(nil)
		u.window.Close()
		u.app.Quit()
	}
	if u.timer.active {
		u.window.Show()
		dialog.ShowConfirm("Cancel timer and quit?", "Quitting cancels the timer. Use Hide to tray to keep it running.", func(ok bool) {
			if ok {
				quit()
			}
		}, u.window)
		return
	}
	quit()
}

func (u *timerUI) setupTray() {
	if u.tray == nil {
		return
	}
	u.trayState = fyne.NewMenuItem("Nothing scheduled", nil)
	u.trayState.Disabled = true
	u.trayPause = fyne.NewMenuItem("Pause", u.togglePause)
	u.trayExtra = fyne.NewMenuItem("Add 5 minutes", func() { u.extend(5 * time.Minute) })
	u.trayStop = fyne.NewMenuItem("Cancel timer", u.cancelTimer)
	quit := fyne.NewMenuItem("Quit", u.requestQuit)
	quit.IsQuit = true
	u.trayMenu = fyne.NewMenu("Shutdown Timer",
		fyne.NewMenuItem("Show timer", func() { u.window.Show(); u.window.RequestFocus() }),
		u.trayState, fyne.NewMenuItemSeparator(),
		u.trayPause, u.trayExtra, u.trayStop, fyne.NewMenuItemSeparator(),
		quit,
	)
	u.tray.SetSystemTrayIcon(u.app.Icon())
	u.tray.SetSystemTrayWindow(u.window)
	u.tray.SetSystemTrayMenu(u.trayMenu)
	u.refresh(time.Now())
}

func (u *timerUI) refreshTray(text, pauseLabel string) {
	if u.trayMenu == nil {
		return
	}
	if u.trayState.Label == text && u.trayPause.Label == pauseLabel && u.trayStop.Disabled == !u.timer.active {
		return
	}
	u.trayState.Label = text
	u.trayPause.Label = pauseLabel
	for _, item := range []*fyne.MenuItem{u.trayPause, u.trayExtra, u.trayStop} {
		item.Disabled = !u.timer.active
	}
	u.trayMenu.Refresh()
}
