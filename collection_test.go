package nested

import (
	"testing"
	"time"
)

func TestCollection(t *testing.T) {

	co := Collection{}

	// A new collection is not ready.
	s, e := co.GetState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)

	// Add services.
	s0, s1 := &Monitor{}, &Monitor{}
	co.Add(s0)
	co.Add(s1)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)

	// One service is ready.
	s0.SetState(Ready, nil)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)

	// Both services are ready.
	s1.SetState(Ready, nil)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetState()
	assertEqual(t, Ready, s)
	assertEqual(t, nil, e)

	assertEqual(t, 2, co.StateCount(Ready))
	assertEqual(t, 0, co.StateCount(NotReady))
	assertEqual(t, 0, co.StateCount(Stopped))

	// One service is stopped.
	s0.Stop()
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetState()
	assertEqual(t, Stopped, s)
	assertEqual(t, nil, e)

	assertEqual(t, 1, co.StateCount(Ready))
	assertEqual(t, 0, co.StateCount(NotReady))
	assertEqual(t, 1, co.StateCount(Stopped))

	// One service is stopped, and the other is not ready.
	s1.SetState(NotReady, nil)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetState()
	assertEqual(t, Stopped, s)
	assertEqual(t, nil, e)

	// Stop all services.
	co.Stop()
	assertEqual(t, 0, co.StateCount(Ready))
	assertEqual(t, 0, co.StateCount(NotReady))

	// We also have no stopped services because the service list has been emptied.
	assertEqual(t, 0, co.StateCount(Stopped))
}
