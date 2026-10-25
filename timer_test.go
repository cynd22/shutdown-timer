package main

import (
	"testing"
	"time"
)

func TestParseWhen(t *testing.T) {
	now := time.Date(2026, 10, 25, 20, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		input string
		want  time.Duration
	}{
		{"45", 45 * time.Minute}, {"90m", 90 * time.Minute},
		{"1h30m", 90 * time.Minute}, {" 1H 30M ", 90 * time.Minute},
		{"45s", 45 * time.Second}, {"0.5s", 500 * time.Millisecond},
		{"23:30", 210 * time.Minute}, {"19:00", 23 * time.Hour},
		{"20:00", 24 * time.Hour}, {"525600", maxDelay},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseWhenAt(tt.input, now)
			if err != nil || got != tt.want {
				t.Fatalf("got %v, %v; want %v", got, err, tt.want)
			}
		})
	}
	for _, input := range []string{"", "0", "-1", "-45s", "0s", "25:00", "12:60", "tomorrow", "525601", "9223372036854775807", "9223372036854775808", "999999999999999h"} {
		if _, err := parseWhenAt(input, now); err == nil {
			t.Errorf("accepted invalid input %q", input)
		}
	}
}

func TestClockTimeAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("Australia/Melbourne")
	if err != nil {
		t.Fatal(err)
	}
	// Daylight saving starts overnight: tomorrow at 10 is 23 hours away.
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, loc)
	delay, err := parseWhenAt("10:00", now)
	if err != nil || delay != 23*time.Hour {
		t.Fatalf("got %v, %v; want 23h", delay, err)
	}
	// The local time 02:30 is skipped on the next day.
	now = time.Date(2026, 10, 3, 23, 0, 0, 0, loc)
	if _, err := parseWhenAt("02:30", now); err == nil {
		t.Fatal("accepted nonexistent clock time")
	}
	// Daylight saving ends overnight: next 10:00 is 25 hours away.
	now = time.Date(2026, 4, 4, 10, 0, 0, 0, loc)
	delay, err = parseWhenAt("10:00", now)
	if err != nil || delay != 25*time.Hour {
		t.Fatalf("got %v, %v; want 25h", delay, err)
	}
}

func TestTimerWarningAndExpiryOnce(t *testing.T) {
	now := time.Now()
	var timer timerState
	timer.start(now, 2*time.Minute, shutdown)
	if warn, fire := timer.advance(now.Add(time.Minute), time.Minute); !warn || fire {
		t.Fatal("expected warning before expiry")
	}
	if warn, fire := timer.advance(now.Add(90*time.Second), time.Minute); warn || fire {
		t.Fatal("warning repeated")
	}
	// Simulate resuming after the deadline. Only one action is dispatched.
	if warn, fire := timer.advance(now.Add(time.Hour).Round(0), time.Minute); warn || !fire {
		t.Fatal("expected one action on wake")
	}
	if warn, fire := timer.advance(now.Add(2*time.Hour), time.Minute); warn || fire {
		t.Fatal("action repeated")
	}
}

func TestTimerPauseExtendResumeAndReplace(t *testing.T) {
	now := time.Now()
	var timer timerState
	timer.start(now, time.Minute, shutdown)
	timer.togglePause(now.Add(20 * time.Second))
	if !timer.paused || timer.remaining(now.Add(time.Hour)) != 40*time.Second {
		t.Fatal("paused timer did not freeze")
	}
	if warn, fire := timer.advance(now.Add(time.Hour), time.Minute); warn || fire {
		t.Fatal("paused timer advanced")
	}
	timer.extend(now.Add(time.Hour), 5*time.Minute)
	timer.togglePause(now.Add(time.Hour))
	if timer.paused || timer.remaining(now.Add(time.Hour)) != 340*time.Second {
		t.Fatal("resume lost the paused extension")
	}
	timer.start(now, 2*time.Hour, restart)
	if timer.action != restart || timer.paused || timer.remaining(now) != 2*time.Hour {
		t.Fatal("replacement retained stale state")
	}
	timer.extend(now, 15*time.Minute)
	if timer.remaining(now) != 135*time.Minute {
		t.Fatal("running extension failed")
	}
	timer = timerState{} // cancellation
	if warn, fire := timer.advance(now.Add(3*time.Hour), time.Minute); warn || fire {
		t.Fatal("cancelled timer fired")
	}
}

func TestSnoozeRearmsWarning(t *testing.T) {
	now := time.Now()
	var timer timerState
	timer.start(now, 30*time.Second, shutdown)
	timer.advance(now, time.Minute)
	timer.extend(now, 5*time.Minute)
	if warn, _ := timer.advance(now, time.Minute); warn {
		t.Fatal("warning before the snoozed deadline")
	}
	if warn, _ := timer.advance(now.Add(5*time.Minute), time.Minute); !warn {
		t.Fatal("extension did not rearm warning")
	}
}

func TestCountdownRoundsUp(t *testing.T) {
	for _, tt := range []struct {
		delay time.Duration
		want  string
	}{{time.Nanosecond, "00:00:01"}, {time.Hour + 2*time.Minute, "01:02:00"}, {-time.Second, "00:00:00"}} {
		if got := fmtDur(tt.delay); got != tt.want {
			t.Errorf("fmtDur(%v) = %q; want %q", tt.delay, got, tt.want)
		}
	}
}
