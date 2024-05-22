package nested

import (
	"errors"
	"testing"
	"time"
)

func TestCollection(t *testing.T) {

	co := Collection{}

	// A new collection is initializing
	assertEqual(t, Initializing, co.GetState())
	assertEqual(t, nil, co.Err())
	assertEqual(t, map[string]bool{}, co.Up())

	// Add services.
	s0, s1 := &Monitor{}, &Monitor{}
	co.Add("service 0", s0)
	co.Add("service 1", s1)
	co.Run()
	time.Sleep(10 * time.Millisecond)
	assertEqual(t, Initializing, co.GetState())
	assertEqual(t, nil, co.Err())
	assertEqual(t, map[string]bool{"service 0": false, "service 1": false}, co.Up())

	// Can't add another service 0.
	assertPanic(t, func() { co.Add("service 0", s0) }, `add: label "service 0" already in use`)

	// One service is ready.
	s0.SetReady()
	time.Sleep(10 * time.Millisecond)
	assertEqual(t, Initializing, co.GetState())
	assertEqual(t, nil, co.Err())
	assertEqual(t, map[string]bool{"service 0": true, "service 1": false}, co.Up())

	// Both services are ready.
	s1.SetReady()
	time.Sleep(10 * time.Millisecond)
	assertEqual(t, Ready, co.GetState())
	assertEqual(t, nil, co.Err())
	assertEqual(t, map[string]bool{"service 0": true, "service 1": true}, co.Up())

	// One service is stopped.
	s0.Stop()
	time.Sleep(10 * time.Millisecond)
	assertEqual(t, Error, co.GetState())
	assertEqual(t, ErrStoppedServices, co.Err())
	assertEqual(t, map[string]bool{"service 0": false, "service 1": true}, co.Up())

	// One service is stopped, and the other is not ready.
	nr := errors.New("not ready")
	s1.SetError(nr)
	time.Sleep(10 * time.Millisecond)
	assertEqual(t, Error, co.GetState())
	assertEqual(t, error(CollectionError{Errors: map[string]error{"service 1": nr}}), co.Err())
	assertEqual(t, map[string]bool{"service 0": false, "service 1": false}, co.Up())

	// Stop all services.
	co.Stop()
	assertEqual(t, map[string]bool{"service 0": false, "service 1": false}, co.Up())
}
