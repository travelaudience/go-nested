package nested

type State int8

const (
	NotReady State = iota
	Ready
	Stopped
)

var names = map[State]string{
	NotReady: "not ready",
	Ready:    "ready",
	Stopped:  "stopped",
}

func (s State) String() string {
	return names[s]
}

type Service interface {
	// GetState returns the current state of the service.
	GetState() State
	// GetFullState returns the current state and error state of the service.
	GetFullState() (State, error)
	// Stop stops the service, and releases all resources.  After sending the final update to the stopped state,
	// all subscriptions are unsubscribed.  Future calls to GetState() will always return Stopped.
	Stop()
	// Subscribe starts sending all state changes to the channel provided.  The ID must unique.  Subscribe panics
	// if the ID is already subscribed.
	Subscribe(id string, channel chan<- Notification)
	// Unsubscribe stops sending notifications.  The caller must provide the same ID as was provided in the call
	// to Subscribe().  Repeated calls to Unsubscribed() with the same ID are ignored.  Calls to Unscrubscribe()
	// with an unknown ID are also ignored.
	Unsubscribe(id string)
}

type Notification struct {
	// The ID as provided by the call to Subscribe()
	ID string
	// The new state
	State State
	// The new error state
	Error error
}
