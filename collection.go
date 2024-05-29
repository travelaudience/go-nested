package nested

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// A Collection monitors multiple services and keeps track of the overall state.  The overall state is defined as:
//   - Ready if ALL of the services are ready.
//   - Stopped if ALL of the services are stopped.
//   - Error if ANY of the services are erroring.
//   - Error if some (but not all) of the services are stopped.
//   - Initializing of ANY of the services are initializing (and none are erroring).
//
// A Collection implements the Service interface.
//
// Services to be monitored are added using the Add() method.  Services cannot be removed once added.
//
// To start monitoring, the caller must invoke the Run() method.  Only when Run has been called AND all of the services
// have finished initialization will the collection change its state.  Services should not be added after calling Run().
//
// An empty Collection is ready to use and in the Initializing state.  A Collection must not be copied after first use.
type Collection struct {
	Monitor
	sync.Mutex
	services map[string]Service
	running  bool
	id       string // random id to distinguish this from other collections when registering observers
}

// Verifies that a Collection implements the Service interface.
var _ Service = &Collection{}

// A CollectionError is returned by the collections Err() method when any of the services are erroring.  It can be
// inspected for details of the errors from each service.
type CollectionError struct {
	// Errors contains the error descriptions from each erroring service, indexed by label.  Only erroring services are included.
	Errors map[string]error
}

// An ErrStoppedServices error is returned by the collections Err() method when no services are erroring and some (but
// not all) of the monitored services are stopped.  It normally indicates that we're in the process of shutting down.
var ErrStoppedServices = errors.New("there are stopped services")

// Error returns the error descriptions from all erroring services in a multi-line string.
func (ce CollectionError) Error() string {
	msgs := make([]string, 0, len(ce.Errors))
	for id, err := range ce.Errors {
		msgs = append(msgs, id+": "+err.Error())
	}
	sort.Strings(msgs)
	return strings.Join(msgs, "\n")
}

// Add adds a service to be monitored.  Panics if the label has already been used in this collection.
func (c *Collection) Add(label string, s Service) {
	c.Lock()
	defer c.Unlock()

	// Initialize the maps if this is the first service to be added.
	if c.services == nil {
		c.services = make(map[string]Service)
		c.id = strconv.FormatUint(rand.Uint64(), 16)
	} else {
		// Otherwise check that we're not reusing a label.
		if _, ok := c.services[label]; ok {
			panic(fmt.Sprintf("add: label %q already in use", label))
		}
	}

	c.services[label] = s

	// Just in case someone adds a service to a running collection, make sure we get its events.  The alternative would
	// be to disallow adding the service in the first place, but we don't want to do that.
	if c.running {
		s.RegisterCallback(c.id, c.stateChanged)
	}
}

// Run starts monitoring the added services.  The collection remains in the Initializing state until all of the
// monitored services are finished initializing.
//
// Calling Run on an already running collection has no effect.
func (c *Collection) Run() {
	defer c.stateChanged(Event{})
	c.Lock()
	defer c.Unlock()
	for _, s := range c.services {
		s.RegisterCallback(c.id, c.stateChanged)
	}
}

// Up returns a map whose keys are the labels of all the currently monitored services and whose values are true if
// the service is ready and false otherwise.
func (c *Collection) Up() map[string]bool {
	up := make(map[string]bool)
	c.Lock()
	defer c.Unlock()
	for label, service := range c.services {
		up[label] = service.GetState() == Ready
	}
	return up
}

// Stop stops the collection and all monitored services and releases all of the resources.  Neither the collection nor
// any of the services should be used after calling stop.
func (c *Collection) Stop() {

	// Initialize the wait group first so that wg.Wait() runs after the lock is released.  That way, if we block
	// on any of the Stop() calls, we do so without holding the lock.
	wg := sync.WaitGroup{}
	defer wg.Wait()

	c.Lock()
	defer c.Unlock()

	wg.Add(len(c.services))
	for _, service := range c.services {
		go func(s Service) {
			s.Stop()
			wg.Done()
		}(service)
	}
}

// stateChanged updates the state of the collection according to the states of all of the monitored services.  No update is
// done if any of the services are still initializing.
func (c *Collection) stateChanged(_ Event) {

	c.Lock()
	defer c.Unlock()

	allStopped := true
	anyStopped := false
	errs := make(map[string]error)

	if len(c.services) == 0 {
		return
	}

	for id, s := range c.services {
		switch s.GetState() {
		case Initializing:
			return
		case Ready:
			allStopped = false
		case Error:
			errs[id] = s.Err()
			allStopped = false // not actually needed, since we check for errors first
		case Stopped:
			anyStopped = true
		}
	}

	if len(errs) > 0 {
		c.Monitor.SetError(CollectionError{Errors: errs})
		return
	}

	if allStopped {
		c.Monitor.Stop()
		return
	}

	if anyStopped {
		c.Monitor.SetError(ErrStoppedServices)
		return
	}

	c.Monitor.SetReady()
}
