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

	// Set to Ready.
	mon.SetReady()
	assertEqual(t, Ready, mon.GetState())
	assertEqual(t, nil, mon.Err())

	// Set to Error.
	reason := errors.New("some reason")
	mon.SetError(reason)
	assertEqual(t, Error, mon.GetState())
	assertEqual(t, reason, mon.Err())

	// Set Ready again.  Previous error can still be retrieved.
	mon.SetReady()
	assertEqual(t, Ready, mon.GetState())
	assertEqual(t, reason, mon.Err())

	// Stop.
	mon.Stop()
	assertEqual(t, Stopped, mon.GetState())

	// Can't restart.
	assertPanic(t, func() { mon.SetReady() }, "cannot transition from stopped state")
	assertPanic(t, func() { mon.SetError(reason) }, "cannot transition from stopped state")
}

func TestMonitorNotifications(t *testing.T) {

	mon := Monitor{}
	ch := make(testObserver, 1)
	mon.Register(ch)

	// Set to Ready.
	mon.SetReady()
	n := assertReceived(t, ch)
	assertEqual(t, Initializing, n.OldState)
	assertEqual(t, Ready, n.NewState)
	assertEqual(t, nil, n.Error)

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

	// Set ready again.
	mon.SetReady()
	n = assertReceived(t, ch)
	assertEqual(t, Error, n.OldState)
	assertEqual(t, Ready, n.NewState)
	assertEqual(t, nil, n.Error)

	// Stop.
	mon.Stop()
	n = assertReceived(t, ch)
	assertEqual(t, Ready, n.OldState)
	assertEqual(t, Stopped, n.NewState)
	assertEqual(t, nil, n.Error)

	// Stop again, and there's not an additional notification.
	mon.Stop()
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	close(ch)
}

func TestDeregister(t *testing.T) {

	mon := Monitor{}
	ch := make(testObserver, 1)

	// Deregistering something that doesn't exist is not an error.
	mon.Deregister(ch)

	mon.Register(ch)

	// Set to ready.
	mon.SetReady()
	n := assertReceived(t, ch)
	assertEqual(t, n.OldState, Initializing)
	assertEqual(t, n.NewState, Ready)
	assertEqual(t, n.Error, nil)

	mon.Deregister(ch)

	// No more notifications.
	mon.Stop()
	if len(ch) > 0 {
		t.Error("unexpected notification")
	}

	close(ch)
}

type testObserver chan Event

func (ch testObserver) OnNotify(ev Event) { ch <- ev }
