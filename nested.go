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

// The Service interface defines the behavior of a nested service.
type Service interface {
	// GetState returns the current state of the service.
	GetState() State
	// Err returns the most recent error condition.  Returns nil if the service has never been in the Error state.
	Err() error
	// ErrCount returns the number of consecutive Error states.  Returns 0 if the service is not in the Error state.
	ErrCount() int
	// Stop stops the service and releases all resources.  Stop should not return until the service shutdown is complete.
	Stop()
	// RegisterCallback registers a function which will be called any time there is a state change.  Returns a token
	// that can be used to deregister it later.
	RegisterCallback(f func(Event)) Token
	// Deregister removes a registered callback.  Does nothing if there is no callback registered with the provided token.
	DeregisterCallback(Token)
}

// A Token identifies a registered callback so that it can later be deregistered.
type Token uint32
