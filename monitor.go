package nested

import (
	"sync"
)

// A Monitor is a basic implementation of the nested service finite state machine.
//
// An empty Monitor is ready to use and in the Not Ready state.  A Monitor must not be copied after first use.
type Monitor struct {
	sync.Mutex
	state     State // current state
	err       error // current error state, if the state is not ready
	observers map[Observer]struct{}
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

// Err returns the error from the most recent Err state, or nil if the Monitor has never been in the error state.
func (m *Monitor) Err() error {
	m.Lock()
	defer m.Unlock()
	return m.err
}

// Stop sets the service to stopped.  If there are registered observers, all observers are called before returning.
func (m *Monitor) Stop() {
	m.setState(Stopped, nil)
}

// Register registers an observer, whose OnNotify method will be called any time there is a state change.  Does nothing
// if the observer is already registered.
func (m *Monitor) Register(o Observer) {
	m.Lock()
	defer m.Unlock()
	if m.observers == nil {
		m.observers = make(map[Observer]struct{})
	}
	m.observers[o] = struct{}{}
}

// Deregister removes a registered observer.  Does nothing if the observer is not registered.
func (m *Monitor) Deregister(o Observer) {
	m.Lock()
	defer m.Unlock()
	delete(m.observers, o)
}

// SetReady sets the monitor state to Ready.  If there are registered observers, all observers are called before returning.
// Panics if the monitor is already stopped.
func (m *Monitor) SetReady() {
	m.setState(Ready, nil)
}

// SetReady sets the monitor state to Error.  If there are registered observers, all observers are called before returning.
// Panics if the monitor is already stopped.
func (m *Monitor) SetError(err error) {
	m.setState(Error, err)
}

func (m *Monitor) setState(newState State, newErr error) {

	// Initialize the wait group first so that wg.Wait() runs after the lock is released.  That way, if we block
	// on any of the observer callbacks, we do so without holding the lock.
	var wg sync.WaitGroup
	defer wg.Wait()

	m.Lock()
	defer m.Unlock()

	if newState == m.state && !(newState == Error && newErr != m.err) {
		return // nothing to do
	}

	if m.state == Stopped {
		panic("cannot transition from stopped state")
	}

	ev := Event{
		OldState: m.state,
		NewState: newState,
		Error:    newErr,
	}

	m.state = newState
	if newState == Error {
		m.err = newErr
	}

	// Notify all observers.
	wg.Add(len(m.observers))
	for o := range m.observers {
		// Run these in the background so as not to block while holding the lock.
		go func(o Observer) {
			o.OnNotify(ev)
			wg.Done()
		}(o)
	}
}
