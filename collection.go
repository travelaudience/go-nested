package nested

import (
	"math/rand"
	"strconv"
	"sync"
)

// A Collection monitors multiple services and keeps track of the overall state.  The overall state is defined as:
//   - Ready if all of the services are ready.
//   - Stopped if ANY of the services are stopped.
//   - Not Ready otherwise.
//
// A Collection implements the Service interface but does not set the error states.
//
// Services to be monitored are added using the Add() method.  Services cannot be removed once added.
//
// An empty Collection is ready to use and in the Not Ready state.  A Collection must not be copied after first use.
type Collection struct {
	Monitor
	sync.Mutex
	services map[string]Service
	id       string
	updates  chan Notification
}

// Verifies that a Monitor implements the Service interface.
var _ Service = &Collection{}

// Add adds a service to be monitored.  Panics if the service has already been added.  Panics if the label has been
// used already for another service.
func (c *Collection) Add(label string, s Service) {
	c.Lock()
	defer c.Unlock()

	// Initialize the update channel if this is the first service to be added.
	if c.services == nil {
		c.services = make(map[string]Service)
		c.updates = make(chan Notification)
		go func() {
			for range c.updates {
				c.Monitor.SetState(c.getOverallState(), nil)
			}
		}()
		// Using the same ID to subscribe to all monitored services means that Subscribe will panic below if a service
		// is added twice.
		c.id = "collection-" + strconv.Itoa(rand.Int())
	} else {
		// Otherwise check that we're not reusing a label.
		if _, ok := c.services[label]; ok {
			panic("add: label " + label + " already in use")
		}
	}

	c.services[label] = s
	s.Subscribe(c.id, c.updates)

	// Trigger an update to include the state of the newly added service.
	go func() {
		c.updates <- Notification{}
	}()
}

// StateCount returns the number of monitored services currently in the given state.
func (c *Collection) StateCount(state State) int {
	c.Lock()
	defer c.Unlock()

	var n int
	for _, service := range c.services {
		if service.GetState() == state {
			n++
		}
	}
	return n
}

// Up returns a map containing the labels of all the currently monitored services and an indication of whether each is
// in the ready state (true) or not (false).
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

	// Start stopping all of the member services, and then release the lock.
	u := func() chan Notification {

		// Initialize the wait group first so that wg.Wait() runs after the lock is released.  That way, if we block
		// on any of the Stop() calls, we do so without holding the lock.
		wg := sync.WaitGroup{}
		defer wg.Wait()

		c.Lock()
		defer c.Unlock()

		wg.Add(len(c.services))
		for _, service := range c.services {
			// Unsubscribe first so that we can close the notifications channel.  Note that a side effect of
			// unsubscribing here is that we need to explicitly set the monitor to stopped when we're done.
			service.Unsubscribe(c.id)
			go func(s Service) {
				s.Stop()
				wg.Done()
			}(service)
		}
		c.services = nil

		// Return the update channel so that we don't have to grab the lock again to get it.
		return c.updates
	}()

	// Close the update channel to release the goroutine in Add() above.  If u is nil, that means that this collection
	// hasn't been used, which is unexpected but not our concern.
	if u != nil {
		close(u)
	}

	// Need to explicitly set the monitor to stopped, since we unsubscribed already above.
	c.Monitor.Stop()
}

// getOverallState computes the overall state of the collection: ready if all of the services are ready, stopped
// if any of the services are stopped, and not ready otherwise.  getOverallState should not be called on an empty
// collection, as it will give the incorrect state.
func (c *Collection) getOverallState() State {
	c.Lock()
	defer c.Unlock()

	we := State(Ready)
	for _, service := range c.services {
		switch service.GetState() {
		case Stopped:
			return Stopped
		case NotReady:
			we = NotReady
		}
	}
	return we
}
