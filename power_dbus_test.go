package main

import (
	"bufio"
	"context"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// This fixture owns an isolated bus; it cannot reach the desktop's real
// shutdown service. The command fallback below is always a failing stub.
type mockShutdownService struct {
	calls atomic.Int32
	deny  atomic.Bool
}

func (s *mockShutdownService) LogoutAndShutdown() *dbus.Error {
	s.calls.Add(1)
	if s.deny.Load() {
		return dbus.NewError("org.freedesktop.DBus.Error.AccessDenied", []interface{}{"denied by mock"})
	}
	return nil
}

func TestAcknowledgedKDERequestOverPrivateBus(t *testing.T) {
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("dbus-daemon unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	daemon := exec.CommandContext(ctx, "dbus-daemon", "--session", "--nofork", "--nopidfile", "--print-address=1")
	stdout, err := daemon.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { daemon.Process.Kill(); daemon.Wait() }()
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("private D-Bus daemon did not start")
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", scanner.Text())
	server, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	service := &mockShutdownService{}
	if err := server.ExportWithMap(service, map[string]string{"LogoutAndShutdown": "logoutAndShutdown"}, "/Shutdown", "org.kde.Shutdown"); err != nil {
		t.Fatal(err)
	}
	if reply, err := server.RequestName("org.kde.Shutdown", dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("could not own mock service: %v, %v", reply, err)
	}
	client, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	e := actionExecutor{kde: true,
		session: func(ctx context.Context, dest string, path dbus.ObjectPath, method string, args ...interface{}) error {
			return client.Object(dest, path).CallWithContext(ctx, method, 0, args...).Err
		},
		command: func(context.Context, ...string) error {
			t.Error("attempted to bypass the mock KDE service")
			return context.Canceled
		},
	}
	if err := e.execute(ctx, shutdown); err != nil || service.calls.Load() != 1 {
		t.Fatalf("request failed: %v, calls %d", err, service.calls.Load())
	}
	service.deny.Store(true)
	if err := e.execute(ctx, shutdown); err == nil || service.calls.Load() != 2 {
		t.Fatalf("D-Bus error was not received: %v, calls %d", err, service.calls.Load())
	}
}
