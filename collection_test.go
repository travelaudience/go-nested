package nested

import (
	"errors"
	"testing"
	"time"
)

func TestCollection(t *testing.T) {

	co := Collection{}

	// A new collection is not ready.
	s, e := co.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{}, co.Up())

	// Add services.
	s0, s1 := &Monitor{}, &Monitor{}
	co.Add("service 0", s0)
	co.Add("service 1", s1)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{"service 0": false, "service 1": false}, co.Up())

	// One service is ready.
	s0.SetState(Ready, nil)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{"service 0": true, "service 1": false}, co.Up())

	// Both services are ready.
	s1.SetState(Ready, nil)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetFullState()
	assertEqual(t, Ready, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, true, co.IsUp())
	assertEqual(t, map[string]bool{"service 0": true, "service 1": true}, co.Up())

	assertEqual(t, 2, co.StateCount(Ready))
	assertEqual(t, 0, co.StateCount(NotReady))
	assertEqual(t, 0, co.StateCount(Stopped))

	// One service is stopped.
	s0.Stop()
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetFullState()
	assertEqual(t, Stopped, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{"service 0": false, "service 1": true}, co.Up())

	assertEqual(t, 1, co.StateCount(Ready))
	assertEqual(t, 0, co.StateCount(NotReady))
	assertEqual(t, 1, co.StateCount(Stopped))

	// One service is stopped, and the other is not ready.
	s1.SetState(NotReady, nil)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetFullState()
	assertEqual(t, Stopped, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{"service 0": false, "service 1": false}, co.Up())

	// Stop all services.
	co.Stop()
	assertEqual(t, 0, co.StateCount(Ready))
	assertEqual(t, 0, co.StateCount(NotReady))
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{}, co.Up())

	// We also have no stopped services because the service list has been emptied.
	assertEqual(t, 0, co.StateCount(Stopped))
}

func TestCollection2(t *testing.T) {

	co := Collection{}

	// A new collection is not ready.
	s, e := co.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)

	// Add two services; one is ready, one isn't.
	s0, s1 := &Monitor{}, &Monitor{}
	s0.SetState(Ready, nil)
	co.Add("service 0", s0)
	s1.SetState(NotReady, errors.New("oh, no!"))
	co.Add("service 1", s1)
	time.Sleep(10 * time.Millisecond)
	s, e = co.GetFullState()
	assertEqual(t, NotReady, s)
	assertEqual(t, nil, e)
	assertEqual(t, s, co.GetState())
	assertEqual(t, false, co.IsUp())
	assertEqual(t, map[string]bool{"service 0": true, "service 1": false}, co.Up())
}
