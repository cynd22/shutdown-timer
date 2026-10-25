package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestKDESessionActions(t *testing.T) {
	for action, method := range map[powerAction]string{shutdown: "logoutAndShutdown", restart: "logoutAndReboot", logout: "logout"} {
		t.Run(string(action), func(t *testing.T) {
			calls := 0
			e := actionExecutor{kde: true,
				session: func(ctx context.Context, dest string, path dbus.ObjectPath, name string, args ...interface{}) error {
					calls++
					if dest != "org.kde.Shutdown" || path != "/Shutdown" || name != "org.kde.Shutdown."+method || len(args) != 0 {
						t.Fatalf("wrong KDE request: %s %s %s %v", dest, path, name, args)
					}
					if _, ok := ctx.Deadline(); !ok {
						t.Fatal("unbounded D-Bus request")
					}
					return nil
				}, command: func(context.Context, ...string) error { t.Fatal("bypassed KDE"); return nil },
			}
			if err := e.execute(context.Background(), action); err != nil || calls != 1 {
				t.Fatalf("got %v, %d calls", err, calls)
			}
		})
	}
}

func TestKDEFailureNeverForcesSystemShutdown(t *testing.T) {
	for _, failure := range []error{
		context.DeadlineExceeded, errors.New("session bus disconnected"),
		dbus.Error{Name: "org.freedesktop.DBus.Error.AccessDenied"},
		dbus.Error{Name: "org.freedesktop.DBus.Error.NoReply"},
	} {
		e := actionExecutor{kde: true,
			session: func(context.Context, string, dbus.ObjectPath, string, ...interface{}) error { return failure },
			command: func(context.Context, ...string) error {
				t.Fatal("ran system fallback on ambiguous KDE failure")
				return nil
			},
		}
		if err := e.execute(context.Background(), shutdown); err == nil {
			t.Fatalf("ignored %v", failure)
		}
	}
}

func TestLegacyKDEFallback(t *testing.T) {
	for action, typeID := range map[powerAction]int32{shutdown: 2, restart: 1, logout: 0} {
		calls := 0
		e := actionExecutor{kde: true,
			session: func(_ context.Context, dest string, path dbus.ObjectPath, name string, args ...interface{}) error {
				calls++
				if calls == 1 {
					return dbus.Error{Name: "org.freedesktop.DBus.Error.ServiceUnknown"}
				}
				if dest != "org.kde.ksmserver" || path != "/KSMServer" || name != "org.kde.KSMServerInterface.logout" || !reflect.DeepEqual(args, []interface{}{int32(0), typeID, int32(2)}) {
					t.Fatalf("wrong legacy request: %s %s %s %v", dest, path, name, args)
				}
				return nil
			}, command: func(context.Context, ...string) error { t.Fatal("bypassed legacy KDE"); return nil },
		}
		if err := e.execute(context.Background(), action); err != nil || calls != 2 {
			t.Fatalf("got %v, %d calls", err, calls)
		}
	}
}

func TestOriginalShutdownFallbacks(t *testing.T) {
	for _, kde := range []bool{false, true} {
		var commands []string
		e := actionExecutor{kde: kde,
			session: func(context.Context, string, dbus.ObjectPath, string, ...interface{}) error {
				return dbus.Error{Name: "org.freedesktop.DBus.Error.ServiceUnknown"}
			}, command: func(_ context.Context, args ...string) error {
				commands = append(commands, strings.Join(args, " "))
				if len(commands) == 3 {
					return nil
				}
				return errors.New("unavailable")
			},
		}
		if err := e.execute(context.Background(), shutdown); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(commands, []string{"systemctl poweroff", "shutdown now", "poweroff"}) {
			t.Fatalf("changed existing fallback order: %v", commands)
		}
	}
}

func TestPowerCommandsAndTimeout(t *testing.T) {
	for action, command := range map[powerAction]string{restart: "systemctl reboot", suspend: "systemctl suspend", hibernate: "systemctl hibernate"} {
		e := actionExecutor{kde: true,
			session: func(context.Context, string, dbus.ObjectPath, string, ...interface{}) error {
				return dbus.Error{Name: "org.freedesktop.DBus.Error.ServiceUnknown"}
			}, command: func(_ context.Context, args ...string) error {
				if strings.Join(args, " ") != command {
					t.Fatalf("wrong command: %v", args)
				}
				return nil
			},
		}
		if err := e.execute(context.Background(), action); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	e := actionExecutor{command: func(context.Context, ...string) error { calls++; return context.DeadlineExceeded }}
	if err := e.execute(context.Background(), shutdown); err == nil || calls != 1 {
		t.Fatalf("timeout retried: %d, %v", calls, err)
	}
	if err := e.execute(context.Background(), powerAction("bad")); err == nil {
		t.Fatal("accepted unknown action")
	}
	if err := e.execute(context.Background(), logout); err == nil {
		t.Fatal("non-KDE logout should be unsupported")
	}
}

func TestKDEDetection(t *testing.T) {
	t.Setenv("KDE_FULL_SESSION", "")
	t.Setenv("DESKTOP_SESSION", "")
	for value, want := range map[string]bool{"KDE": true, "kde": true, "Plasma": true, "ubuntu:KDE": true, "GNOME": false, "": false} {
		t.Setenv("XDG_CURRENT_DESKTOP", value)
		if isKDE() != want {
			t.Errorf("KDE detection for %q", value)
		}
	}
}
