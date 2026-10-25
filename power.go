package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

type powerAction string

const (
	shutdown  powerAction = "Shut down"
	restart   powerAction = "Restart"
	suspend   powerAction = "Sleep"
	hibernate powerAction = "Hibernate"
	logout    powerAction = "Log out"
)

var actionNames = []string{string(shutdown), string(restart), string(suspend), string(hibernate), string(logout)}

func validAction(action powerAction) bool {
	for _, name := range actionNames {
		if string(action) == name {
			return true
		}
	}
	return false
}

func isKDE() bool {
	for _, desktop := range strings.FieldsFunc(os.Getenv("XDG_CURRENT_DESKTOP"), func(r rune) bool { return r == ':' || r == ';' }) {
		if strings.EqualFold(desktop, "KDE") || strings.EqualFold(desktop, "Plasma") {
			return true
		}
	}
	return strings.EqualFold(os.Getenv("KDE_FULL_SESSION"), "true") || strings.EqualFold(os.Getenv("DESKTOP_SESSION"), "plasma")
}

type sessionCall func(context.Context, string, dbus.ObjectPath, string, ...interface{}) error
type commandCall func(context.Context, ...string) error

type actionExecutor struct {
	kde     bool
	session sessionCall
	command commandCall
}

// Only a missing service/API is safe to fall back from. A timeout, denied
// request or disconnected bus could mean KDE already started closing apps.
// Never bypass its session saving or an application's unsaved-work prompt.
func missingSessionAPI(err error) bool {
	var busErr dbus.Error
	if !errors.As(err, &busErr) {
		return false
	}
	switch busErr.Name {
	case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner",
		"org.freedesktop.DBus.Error.UnknownMethod", "org.freedesktop.DBus.Error.UnknownObject",
		"org.freedesktop.DBus.Error.UnknownInterface":
		return true
	}
	return false
}

func (e actionExecutor) execute(ctx context.Context, action powerAction) error {
	if !validAction(action) {
		return fmt.Errorf("unknown action %q", action)
	}
	if e.kde && (action == shutdown || action == restart || action == logout) {
		method := map[powerAction]string{shutdown: "logoutAndShutdown", restart: "logoutAndReboot", logout: "logout"}[action]
		requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := e.session(requestCtx, "org.kde.Shutdown", "/Shutdown", "org.kde.Shutdown."+method)
		cancel()
		if err == nil {
			return nil
		}
		if !missingSessionAPI(err) {
			return fmt.Errorf("KDE could not accept the %s request: %w. No system command was run; check your desktop session", strings.ToLower(string(action)), err)
		}
		// Plasma 5 installations without org.kde.Shutdown still provide this.
		typeID := map[powerAction]int32{logout: 0, restart: 1, shutdown: 2}[action]
		requestCtx, cancel = context.WithTimeout(ctx, 15*time.Second)
		err = e.session(requestCtx, "org.kde.ksmserver", "/KSMServer", "org.kde.KSMServerInterface.logout", int32(0), typeID, int32(2))
		cancel()
		if err == nil {
			return nil
		}
		if !missingSessionAPI(err) {
			return fmt.Errorf("KDE could not accept the %s request: %w. No system command was run", strings.ToLower(string(action)), err)
		}
	}

	var candidates [][]string
	switch action {
	case shutdown:
		candidates = [][]string{{"systemctl", "poweroff"}, {"shutdown", "now"}, {"poweroff"}}
	case restart:
		candidates = [][]string{{"systemctl", "reboot"}, {"shutdown", "-r", "now"}, {"reboot"}}
	case suspend:
		candidates = [][]string{{"systemctl", "suspend"}}
	case hibernate:
		candidates = [][]string{{"systemctl", "hibernate"}}
	case logout:
		return fmt.Errorf("log out requires a running KDE Plasma session")
	}
	var failures []error
	for _, args := range candidates {
		commandCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := e.command(commandCtx, args...)
		cancel()
		if err == nil {
			return nil
		}
		failures = append(failures, fmt.Errorf("%s: %w", strings.Join(args, " "), err))
		if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
			break // The command might already have submitted the request.
		}
	}
	return fmt.Errorf("couldn't %s:\n%w\n\nCheck your desktop's power permissions and whether this action is supported", strings.ToLower(string(action)), errors.Join(failures...))
}

func executeAction(ctx context.Context, action powerAction) error {
	var conn *dbus.Conn
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	executor := actionExecutor{
		kde: isKDE(),
		session: func(ctx context.Context, dest string, path dbus.ObjectPath, method string, args ...interface{}) error {
			if conn == nil {
				var err error
				conn, err = dbus.ConnectSessionBus()
				if err != nil {
					return err
				}
			}
			return conn.Object(dest, path).CallWithContext(ctx, method, 0, args...).Err
		},
		command: func(ctx context.Context, args ...string) error {
			out, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				return fmt.Errorf("%w %s", err, strings.TrimSpace(string(out)))
			}
			return nil
		},
	}
	return executor.execute(ctx, action)
}
