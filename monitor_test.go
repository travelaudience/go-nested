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

	// A new Monitor is not ready.
	mon := Monitor{}
	s, e := mon.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)

	// Set to ready.
	mon.SetState(Ready, nil)
	s, e = mon.GetFullState()
	assertEqual(t, Ready, s)
	assertEqual(t, nil, e)

	// Set to not ready with a reason.
	reason := errors.New("some reason")
	mon.SetState(NotReady, reason)
	s, e = mon.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, reason, e)

	// Can't set to an undefined state.
	assertPanic(t, func() { mon.SetState(-1, nil) }, "undefined")

	// Set ready again.
	mon.SetState(Ready, nil)
	s, e = mon.GetFullState()
	assertEqual(t, Ready, s)
	assertEqual(t, nil, e)

	// Stop.
	mon.Stop()
	s, e = mon.GetFullState()
	assertEqual(t, Stopped, s)
	assertEqual(t, nil, e)

	// Can't restart.
	assertPanic(t, func() { mon.SetState(Ready, nil) }, "cannot transition from stopped state")
}

func TestMonitor2(t *testing.T) {

	mon := Monitor{}

	// Failure on initialization.
	failure := errors.New("some failure")
	mon.SetState(Stopped, failure)
	s, e := mon.GetFullState()
	assertEqual(t, Stopped, s)
	assertEqual(t, failure, e)

	// Now Stop() should be a no-op
	mon.Stop()
	s, e = mon.GetFullState()
	assertEqual(t, Stopped, s)
	assertEqual(t, failure, e) // note that the error condition is still there
}

func TestMonitorNotifications(t *testing.T) {

	mon := Monitor{}
	ch := make(chan Notification, 1)
	mon.Subscribe("foo", ch)

	// Set to ready.
	mon.SetState(Ready, nil)
	n := assertReceived(t, ch)
	assertEqual(t, Ready, n.State)
	assertEqual(t, nil, n.Error)

	// Set to ready again, and there's not an additional notification.
	mon.SetState(Ready, nil)
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	// Set to not ready with a reason.
	reason := errors.New("some reason")
	mon.SetState(NotReady, reason)
	n = assertReceived(t, ch)
	assertEqual(t, NotReady, n.State)
	assertEqual(t, reason, n.Error)

	// Set ready again.
	mon.SetState(Ready, nil)
	n = assertReceived(t, ch)
	assertEqual(t, Ready, n.State)
	assertEqual(t, nil, n.Error)

	// Stop.
	mon.Stop()
	n = assertReceived(t, ch)
	assertEqual(t, Stopped, n.State)
	assertEqual(t, nil, n.Error)

	// Stop again, and there's not an additional notification.
	mon.Stop()
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	close(ch)
}

func TestUnsubscribe(t *testing.T) {

	mon := Monitor{}

	// Unsubscribing something that doesn't exist is not an error.
	mon.Unsubscribe("bar")

	ch := make(chan Notification, 1)
	mon.Subscribe("foo", ch)

	// Set to ready.
	mon.SetState(Ready, nil)
	n := assertReceived(t, ch)
	assertEqual(t, n.State, Ready)
	assertEqual(t, n.Error, nil)

	mon.Unsubscribe("foo")

	// No more notifications.
	mon.SetState(NotReady, nil)
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	close(ch)
}
