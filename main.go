// Shutdown Timer — a tiny GUI to power off the machine after a delay or at a
// set time. Built with Fyne so it ships as a single self-contained binary.
//
// It runs its own countdown and only calls the system poweroff at the end, so
// on a normal desktop session it needs no root/sudo (logind/polkit lets the
// active user power off). If that's blocked, it falls back to `shutdown now`.
package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ticker drives the live countdown; nil when nothing is scheduled.
var ticker *time.Ticker
var deadline time.Time
var armed bool

func main() {
	a := app.New()
	w := a.NewWindow("Shutdown Timer")
	w.Resize(fyne.NewSize(360, 320))

	status := widget.NewLabel("Nothing scheduled.")
	status.Alignment = fyne.TextAlignCenter

	countdown := widget.NewLabel("")
	countdown.Alignment = fyne.TextAlignCenter
	countdown.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	custom := widget.NewEntry()
	custom.SetPlaceHolder("e.g. 45  or  90m  or  23:30")

	var cancelBtn *widget.Button

	// schedule arms the timer for the given duration and starts the countdown.
	schedule := func(d time.Duration) {
		if d <= 0 {
			dialog.ShowError(fmt.Errorf("that time is in the past"), w)
			return
		}
		stopTimer()
		deadline = time.Now().Add(d)
		armed = true
		status.SetText("Powering off at " + deadline.Format("15:04:05"))
		cancelBtn.Enable()

		ticker = time.NewTicker(time.Second)
		// Render immediately, then once per second.
		render := func() {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				stopTimer()
				countdown.SetText("00:00:00")
				status.SetText("Shutting down…")
				poweroff(w)
				return
			}
			fyne.Do(func() { countdown.SetText(fmtDur(remaining)) })
		}
		render()
		go func() {
			for range ticker.C {
				render()
			}
		}()
	}

	preset := func(label string, d time.Duration) *widget.Button {
		return widget.NewButton(label, func() { schedule(d) })
	}

	presets := container.NewGridWithColumns(4,
		preset("15 min", 15*time.Minute),
		preset("30 min", 30*time.Minute),
		preset("1 hour", time.Hour),
		preset("2 hours", 2*time.Hour),
	)

	startCustom := widget.NewButton("Start", func() {
		d, err := parseWhen(custom.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		schedule(d)
	})

	cancelBtn = widget.NewButton("Cancel shutdown", func() {
		stopTimer()
		status.SetText("Cancelled — nothing scheduled.")
		countdown.SetText("")
		cancelBtn.Disable()
	})
	cancelBtn.Disable()

	content := container.NewVBox(
		widget.NewLabelWithStyle("Shut down after…", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		presets,
		widget.NewSeparator(),
		widget.NewLabel("…or a custom delay / time:"),
		container.NewBorder(nil, nil, nil, startCustom, custom),
		layout.NewSpacer(),
		countdown,
		status,
		cancelBtn,
	)

	w.SetContent(container.NewPadded(content))
	w.SetCloseIntercept(func() {
		if armed {
			dialog.ShowConfirm("Quit?",
				"A shutdown is still scheduled. Quitting cancels it.\nReally quit?",
				func(ok bool) {
					if ok {
						stopTimer()
						w.Close()
					}
				}, w)
			return
		}
		w.Close()
	})
	w.ShowAndRun()
}

func stopTimer() {
	if ticker != nil {
		ticker.Stop()
		ticker = nil
	}
	armed = false
}

// poweroff tries the no-root logind path first, then a couple of fallbacks.
func poweroff(w fyne.Window) {
	candidates := [][]string{
		{"systemctl", "poweroff"},
		{"shutdown", "now"},
		{"poweroff"},
	}
	var lastErr error
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err == nil {
			return
		} else {
			lastErr = fmt.Errorf("%s: %v\n%s", strings.Join(c, " "), err, out)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no shutdown command found")
	}
	fyne.Do(func() {
		dialog.ShowError(fmt.Errorf("couldn't power off:\n%v\n\nYou may need to run from a terminal with sudo.", lastErr), w)
	})
}

func fmtDur(d time.Duration) string {
	total := int(d.Round(time.Second).Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// parseWhen accepts "45" (minutes), "90m", "1h30m", "45s", or "HH:MM".
func parseWhen(text string) (time.Duration, error) {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return 0, fmt.Errorf("enter a delay or time")
	}

	// Absolute clock time HH:MM (today, or tomorrow if already passed).
	if strings.Contains(text, ":") {
		t, err := time.Parse("15:04", text)
		if err != nil {
			return 0, fmt.Errorf("can't read the time %q (use HH:MM)", text)
		}
		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if !target.After(now) {
			target = target.Add(24 * time.Hour)
		}
		return time.Until(target), nil
	}

	// Bare number = minutes.
	if n, err := strconv.Atoi(text); err == nil {
		return time.Duration(n) * time.Minute, nil
	}

	// Duration like 1h30m / 90m / 45s.
	if d, err := time.ParseDuration(text); err == nil {
		return d, nil
	}
	return 0, fmt.Errorf("can't read %q — try 45, 90m, 1h30m, or 23:30", text)
}
