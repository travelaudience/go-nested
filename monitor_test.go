package nested

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func assertPanic(t *testing.T, f func(), msg string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("want panic %q, got no panic", msg)
			return
		}
		got := fmt.Sprint(r)
		if !strings.Contains(got, msg) {
			t.Errorf("want panic containing %q, got panic %q", msg, got)
			return
		}
	}()
	f()
}

func assertReceived[X any](t *testing.T, ch <-chan X) (x X) {
	select {
	case x = <-ch:
	default:
		t.Errorf("no message recevied")
	}
	return
}

func TestMonitor(t *testing.T) {

	// A new Monitor's state is Initializing.
	mon := Monitor{}
	assertEqual(t, Initializing, mon.GetState())
	assertEqual(t, nil, mon.Err())
	assertEqual(t, 0, mon.ErrCount())

	// Set to Ready.
	mon.SetReady()
	assertEqual(t, Ready, mon.GetState())
	assertEqual(t, nil, mon.Err())
	assertEqual(t, 0, mon.ErrCount())

	// Set to Error.
	reason := errors.New("some reason")
	mon.SetError(reason)
	assertEqual(t, Error, mon.GetState())
	assertEqual(t, reason, mon.Err())
	assertEqual(t, 1, mon.ErrCount())

	// Two consecutive errors.
	mon.SetError(reason)
	assertEqual(t, Error, mon.GetState())
	assertEqual(t, reason, mon.Err())
	assertEqual(t, 2, mon.ErrCount())

	// Set Ready again.  Previous error can still be retrieved.
	mon.SetReady()
	assertEqual(t, Ready, mon.GetState())
	assertEqual(t, reason, mon.Err())
	assertEqual(t, 0, mon.ErrCount())

	// Stop.
	mon.Stop()
	assertEqual(t, Stopped, mon.GetState())

	// Can't restart.
	assertPanic(t, func() { mon.SetReady() }, "cannot transition from stopped state")
	assertPanic(t, func() { mon.SetError(reason) }, "cannot transition from stopped state")
}

func TestMonitorNotifications(t *testing.T) {

	mon := Monitor{}
	ch := make(chan Event, 1)
	mon.RegisterCallback(func(ev Event) { ch <- ev })

	// Set to Ready.
	mon.SetReady()
	n := assertReceived(t, ch)
	assertEqual(t, Initializing, n.OldState)
	assertEqual(t, Ready, n.NewState)
	assertEqual(t, nil, n.Error)
	assertEqual(t, 0, n.ErrCount)

	// Set to Ready again, and there's not an additional notification.
	mon.SetReady()
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	// Set to Error.
	reason := errors.New("some reason")
	mon.SetError(reason)
	n = assertReceived(t, ch)
	assertEqual(t, Ready, n.OldState)
	assertEqual(t, Error, n.NewState)
	assertEqual(t, reason, n.Error)
	assertEqual(t, 1, n.ErrCount)

	// Two consecutive errors.
	mon.SetError(reason)
	n = assertReceived(t, ch)
	assertEqual(t, Error, mon.GetState())
	assertEqual(t, reason, mon.Err())
	assertEqual(t, 2, n.ErrCount)

	// Set ready again.
	mon.SetReady()
	n = assertReceived(t, ch)
	assertEqual(t, Error, n.OldState)
	assertEqual(t, Ready, n.NewState)
	assertEqual(t, nil, n.Error)
	assertEqual(t, 0, n.ErrCount)

	// Stop.
	mon.Stop()
	n = assertReceived(t, ch)
	assertEqual(t, Ready, n.OldState)
	assertEqual(t, Stopped, n.NewState)
	assertEqual(t, nil, n.Error)
	assertEqual(t, 0, n.ErrCount)

	// Stop again, and there's not an additional notification.
	mon.Stop()
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	close(ch)
}

func TestDeregister(t *testing.T) {

	mon := Monitor{}
	ch := make(chan Event, 1)

	foo := mon.RegisterCallback(func(ev Event) { ch <- ev })

	// Set to ready.
	mon.SetReady()
	n := assertReceived(t, ch)
	assertEqual(t, n.OldState, Initializing)
	assertEqual(t, n.NewState, Ready)
	assertEqual(t, n.Error, nil)

	mon.DeregisterCallback(foo)

	// No more notifications.
	mon.Stop()
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	// Deregistering again is not an error
	mon.DeregisterCallback(foo)

	close(ch)
}
