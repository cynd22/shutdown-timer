package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const maxDelay = 365 * 24 * time.Hour

// timerState is owned by the UI thread. One app-wide ticker advances it;
// starting, replacing, pausing and cancelling never create goroutines.
type timerState struct {
	active   bool
	paused   bool
	deadline time.Time
	left     time.Duration
	total    time.Duration
	action   powerAction
	warned   bool
}

func (t *timerState) start(now time.Time, delay time.Duration, action powerAction) {
	*t = timerState{
		active: true, deadline: now.Add(delay).Round(0),
		total: delay, action: action,
	}
}

func (t *timerState) remaining(now time.Time) time.Duration {
	if !t.active {
		return 0
	}
	if t.paused {
		return t.left
	}
	// Use wall time so a suspended computer catches up on wake.
	return max(0, t.deadline.Sub(now.Round(0)))
}

func (t *timerState) togglePause(now time.Time) {
	if !t.active {
		return
	}
	if t.paused {
		t.deadline = now.Add(t.left).Round(0)
		t.paused = false
	} else if left := t.remaining(now); left > 0 {
		t.left = left
		t.paused = true
	}
}

func (t *timerState) extend(now time.Time, delay time.Duration) {
	if !t.active {
		return
	}
	left := t.remaining(now)
	delay = min(delay, maxDelay-left)
	if t.paused {
		t.left += delay
	} else {
		t.deadline = now.Add(left + delay).Round(0)
	}
	t.total = min(maxDelay, t.total+delay)
	t.warned = false
}

// advance reports each warning and expiry once. A missed tick never causes
// multiple actions, including after wake or after a timer is replaced.
func (t *timerState) advance(now time.Time, warning time.Duration) (warn, fire bool) {
	if !t.active || t.paused {
		return false, false
	}
	left := t.remaining(now)
	if left <= 0 {
		t.active = false
		return false, true
	}
	if warning > 0 && left <= warning && !t.warned {
		t.warned = true
		return true, false
	}
	return false, false
}

func fmtDur(d time.Duration) string {
	total := int64(math.Ceil(max(0, d).Seconds()))
	return fmt.Sprintf("%02d:%02d:%02d", total/3600, total%3600/60, total%60)
}

// parseWhenAt accepts minutes, Go-style durations or the next local HH:MM.
// Passing now explicitly keeps clock, overflow and DST behavior testable.
func parseWhenAt(text string, now time.Time) (time.Duration, error) {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return 0, fmt.Errorf("enter a delay or time")
	}
	var delay time.Duration
	if strings.Contains(text, ":") {
		clock, err := time.Parse("15:04", text)
		if err != nil {
			return 0, fmt.Errorf("use a 24-hour clock time, such as 23:30")
		}
		day := now
		target := time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, now.Location())
		if !target.After(now) {
			day = now.AddDate(0, 0, 1)
			target = time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, now.Location())
		}
		if target.Hour() != clock.Hour() || target.Minute() != clock.Minute() {
			return 0, fmt.Errorf("%s does not occur on that date because the clocks change; choose another time", text)
		}
		delay = target.Sub(now)
	} else if n, err := strconv.ParseInt(text, 10, 64); err == nil {
		if n <= 0 || n > int64(maxDelay/time.Minute) {
			return 0, fmt.Errorf("enter a delay between 1 minute and 365 days")
		}
		delay = time.Duration(n) * time.Minute
	} else {
		var err error
		delay, err = time.ParseDuration(strings.Join(strings.Fields(text), ""))
		if err != nil {
			return 0, fmt.Errorf("can't read %q — try 45, 90m, 1h30m, 45s, or 23:30", text)
		}
	}
	if delay <= 0 || delay > maxDelay {
		return 0, fmt.Errorf("enter a positive delay of at most 365 days")
	}
	return delay, nil
}
