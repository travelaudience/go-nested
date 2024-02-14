package nested

import (
	"fmt"
	"sync"
)

// A Monitor is a basic implementation of the nested service finite state machine.
//
// An empty Monitor is ready to use and in the Not Ready state.  A Monitor must not be copied after first use.
type Monitor struct {
	sync.Mutex
	state         State // current state
	err           error // current error state
	subscriptions map[string]chan<- Notification
}

// Verifies that a Monitor implements the Service interface.  Note that the Service interface does NOT include the
// SetState() method.  This is by design, as SetState() should only be called by the packiage implementing the service
// and not by consumers of the service.
var _ Service = &Monitor{}

// GetState returns the current state of the service.
func (m *Monitor) GetState() State {
	m.Lock()
	defer m.Unlock()
	return m.state
}

// GetFullState returns the current state and error state of the service.
func (m *Monitor) GetFullState() (State, error) {
	m.Lock()
	defer m.Unlock()
	return m.state, m.err
}

// Stop sets the service to stopped and cancels all subsriptions.  Does nothing if the service is already stopped
// or failed.
func (m *Monitor) Stop() {
	m.setState(Stopped, nil, true)
}

// Subscribe creates a subscription to state changes, and will send all subsequent state changes to the channel provided.
func (m *Monitor) Subscribe(id string, channel chan<- Notification) {
	m.Lock()
	defer m.Unlock()
	if m.subscriptions == nil {
		m.subscriptions = make(map[string]chan<- Notification)
	}
	m.subscriptions[id] = channel
}

// Unsubscribe removes the subscription with the id provided.  Does nothing if the subscription doesn't exist.
func (m *Monitor) Unsubscribe(id string) {
	m.Lock()
	defer m.Unlock()
	delete(m.subscriptions, id)
}

// SetState sets the state and error state.  SetState should only be set by the package implementing the service.
// If there are subscriptions, SetState returns after the notifications are consumed.
//
// SetState can be used to change either the state or the error state, or both.  If a call to SetState results
// in no change, then the result is a no-op.
//
// SetState panics on an attempt to change the state or error state of a stopped service.  (It won't panic if
// there's no change.)
func (m *Monitor) SetState(newState State, newErr error) {
	m.setState(newState, newErr, false)
}

func (m *Monitor) setState(newState State, newErr error, ignoreStopped bool) {

	if _, ok := names[newState]; !ok {
		panic(fmt.Sprintf("state %d is undefined", newState))
	}

	// Initialize the wait group first so that wg.Wait() runs after the lock is released.  That way, if we block
	// on any of the subscription channels, we do so without holding the lock.
	var wg sync.WaitGroup
	defer wg.Wait()

	m.Lock()
	defer m.Unlock()

	if newState == m.state && newErr == m.err {
		return // nothing to do
	}

	if m.state == Stopped {
		if ignoreStopped {
			return
		}
		panic("cannot transition from stopped state")
	}

	m.state, m.err = newState, newErr

	// Notify all subscribers.
	wg.Add(len(m.subscriptions))
	for id, ch := range m.subscriptions {
		// Run these in the background so as not to block while holding the lock.
		go func(id string, ch chan<- Notification) {
			ch <- Notification{ID: id, State: newState, Error: newErr}
			wg.Done()
		}(id, ch)
	}

	if newState == Stopped {
		m.subscriptions = nil
	}
}
