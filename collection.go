package nested

import (
	"strings"
	"sync"
)

// A Collection monitors multiple services and keeps track of the overall state.  The overall state is defined as:
//   - Ready if ALL of the services are ready.
//   - Stopped if ALL of the services are stopped.
//   - Error if ANY of the services are erroring.
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
}

// Verifies that a Collection implements the Service interface.
var _ Service = &Collection{}

// A CallectionError is returned by the collections Err() method when any of the services are erroring.  It can be
// inspected for details of the errors from each service.
type CollectionError struct {
	// Errors contains the error descriptions from each erroring service, indexed by label.  Only erroring services are included.
	Errors map[string]error
}

// Error returns the error descriptions from all erroring services in a multi-line string.
func (ce CollectionError) Error() string {
	var bob strings.Builder
	var sep = ""
	for id, err := range ce.Errors {
		bob.WriteString(sep)
		bob.WriteString(id)
		bob.WriteString(": ")
		bob.WriteString(err.Error())
		sep = "\n"
	}
	return bob.String()
}

// Add adds a service to be monitored.  Panics if the label has already been used in this collection.
func (c *Collection) Add(label string, s Service) {
	c.Lock()
	defer c.Unlock()

	// Initialize the maps if this is the first service to be added.
	if c.services == nil {
		c.services = make(map[string]Service)
	} else {
		// Otherwise check that we're not reusing a label.
		if _, ok := c.services[label]; ok {
			panic("add: label " + label + " already in use")
		}
	}

	c.services[label] = s

	// Just in case someone adds a service to a running collection, make sure we get its events.  The alternative would
	// be to just disallow adding the service, but we don't want to do that.
	if c.running {
		s.Register(c)
	}
}

// Run starts monitoring the added services.  The collection remains in the Initializing state until all of the
// monitored services are finished initializing.
//
// Calling Run on an already running collection has no effect.
func (c *Collection) Run() {
	defer c.OnNotify(Event{})
	c.Lock()
	defer c.Unlock()
	for _, s := range c.services {
		s.Register(c)
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

// OnNotify updates the state of the collection according to the states of all of the monitored services.  No update is
// done if any of the services are still initializing.
//
// OnNotify is used internally as a callback when any monitored service changes state.  It is not necessary to call this
// directly.
func (c *Collection) OnNotify(_ Event) {

	c.Lock()
	defer c.Unlock()

	allStopped := true
	errors := make(map[string]error)

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
			errors[id] = s.Err()
			allStopped = false // not actually needed, since we check for errors first
		}
	}

	if len(errors) > 0 {
		c.Monitor.SetError(CollectionError{Errors: errors})
		return
	}

	if allStopped {
		c.Monitor.Stop()
		return
	}

	c.Monitor.SetReady()
}
