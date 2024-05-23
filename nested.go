package nested

type State int8

const (
	Initializing State = iota
	Ready
	Error
	Stopped
)

var names = map[State]string{
	Initializing: "initializing",
	Ready:        "ready",
	Error:        "error",
	Stopped:      "stopped",
}

func (s State) String() string {
	return names[s]
}

// An event is a single notification of a state change.
type Event struct {
	OldState State
	NewState State
	Error    error // error condition if the new state is Error, nil otherwise
}

// An observer receives notifications of state changes.
type Observer interface {
	OnNotify(Event)
}

// The Service interface defines the behavior of a nested service.
type Service interface {
	// GetState returns the current state of the service.
	GetState() State
	// Err returns the most recent error condition.  Returns nil if the service has never been in the Err state.
	Err() error
	// Stop stops the service and releases all resources.  Stop should not return until the service shutdown is complete.
	Stop()
	// Register registers an observer, whose OnNotify method will be called any time there is a state change.  Does
	// nothing if the observer is already registered.
	Register(Observer)
	// Deregister removes a registered observer.  Does nothing if the observer is not registered.
	Deregister(Observer)
}
