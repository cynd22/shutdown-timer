package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestTimerControlsAndHarmlessDemo(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(theme.DefaultTheme())
	a.SetIcon(fyne.NewStaticResource("icon.png", iconData))
	w := a.NewWindow("Shutdown Timer")
	defer w.Close()
	u := newTimerUI(a, w, true, func(context.Context, powerAction) error {
		t.Fatal("demo attempted a real power action")
		return nil
	})
	if !u.cancel.Disabled() || !u.pause.Disabled() {
		t.Fatal("idle controls should be disabled")
	}
	test.Tap(u.presets[0])
	if !u.timer.active || u.timer.action != shutdown || u.cancel.Disabled() || !u.action.Disabled() {
		t.Fatal("preset did not arm the selected action")
	}
	test.Tap(u.pause)
	if !u.timer.paused || u.pause.Text != "Resume" {
		t.Fatal("pause did not update controls")
	}
	left := u.timer.left
	test.Tap(u.extra5)
	if u.timer.left != left+5*time.Minute {
		t.Fatal("paused extension failed")
	}
	test.Tap(u.pause)
	test.Tap(u.cancel)
	if u.timer.active || !u.cancel.Disabled() || u.action.Disabled() {
		t.Fatal("cancel left stale controls")
	}
	u.action.SetSelected(string(restart))
	u.custom.SetText("2s")
	test.Tap(u.start)
	if !u.timer.active || u.timer.action != restart {
		t.Fatal("custom start lost selected action")
	}
	u.tick(time.Now().Add(3 * time.Second))
	if u.timer.active || u.busy || !strings.Contains(u.status.Text, "Demo complete") || u.countdown.Text != "00:00:00" || u.progress.Value != 1 {
		t.Fatalf("demo expiry failed: %s", u.status.Text)
	}
	u.tick(time.Now().Add(4 * time.Second))
	if !strings.Contains(u.status.Text, "Demo complete") {
		t.Fatal("demo completion was overwritten")
	}
}
