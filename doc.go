// Package nested provides support for the nested service pattern, where a service runs independently from a process
// that uses it but runs on the same machine and is compiled into the same binary.
//
// One important use of nested services is to abstract the details of interfacing with external components.
//
// A nested service is modelled here as a finite state machine, with a Service interface that all nested services
// should implement.
//
// The state machine has the following states:
//   - Initializing.  The service is not ready yet.
//   - Ready.  The service is running normally.
//   - Error.  The service is temporarily unavailable.
//   - Stopped.  The service is permanently unavailable.
//
// The state machine begins in the initializing state.  Once it transitions to one of the other states, it can never
// return to the initializing state.
//
// A state machine in the stopped state cannot change states.
//
// This package also provides a Monitor type, which implements the state machine.  A Monitor can be embedded in
// any service to make it a nested service.
//
// A common pattern is to include a Monitor in the struct that defines the nested service, e.g.
//
//	type MyService struct {
//	    nested.Monitor
//	       ...
//	}
//
// The MyService constructor may either return an initializing service or a fully initialized service.  The MyService
// Stop() method, however, should always wait until the service has stopped completely before returning.
package nested
