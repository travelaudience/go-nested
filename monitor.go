package nested

import (
	"math"
	"math/rand"
	"sync"
)

// A Monitor is a basic implementation of the nested service finite state machine.
//
// An empty Monitor is ready to use and in the Not Ready state.  A Monitor must not be copied after first use.
type Monitor struct {
	sync.Mutex
	state     State // current state
	err       error // current error state, if the state is not ready
	errCount  int   // number of consecutive errors
	callbacks map[Token]func(Event)
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

// ErrCount returns the number of consective errors recorded for this service.  If the service is not in Error state,
// ErrCount returns 0.
func (m *Monitor) ErrCount() int {
	m.Lock()
	defer m.Unlock()
	return m.errCount
}

// Stop sets the service to stopped.  If there are registered observers, all observers are called before returning.
func (m *Monitor) Stop() {
	m.setState(Stopped, nil)
}

// RegisterCallback registers a function which will be called any time there is a state change.  Returns a token that
// can be used to deregister it later.
func (m *Monitor) RegisterCallback(f func(Event)) Token {
	m.Lock()
	defer m.Unlock()
	if m.callbacks == nil {
		m.callbacks = make(map[Token]func(Event))
	}

	// Choose a random token that we haven't used.
	var token Token
	for ok := true; ok; {
		token = Token(rand.Uint32())
		_, ok = m.callbacks[token]
	}

	m.callbacks[token] = f
	return token
}

// Deregister removes a registered callback.  Does nothing if there is no callback registered with the provided token.
func (m *Monitor) DeregisterCallback(token Token) {
	m.Lock()
	defer m.Unlock()
	delete(m.callbacks, token)
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

	if newState == m.state && newState != Error {
		return // nothing to do
	}

	if newState == Error {
		if m.errCount < math.MaxInt {
			m.errCount++
		}
	} else {
		m.errCount = 0
	}

	if m.state == Stopped {
		panic("cannot transition from stopped state")
	}

	ev := Event{
		OldState: m.state,
		NewState: newState,
		Error:    newErr,
		ErrCount: m.errCount,
	}

	m.state = newState
	if newState == Error {
		m.err = newErr
	}

	// Notify all observers.
	wg.Add(len(m.callbacks))
	for _, cb := range m.callbacks {
		// Run these in the background so as not to block while holding the lock.
		go func(f func(Event)) {
			f(ev)
			wg.Done()
		}(cb)
	}
}
