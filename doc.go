// Package nested provides support for the nested service pattern, where a service runs independently from a process
// that uses it but runs on the same machine and is compiled into the same binary.
//
// One important use of nested services is to abstract the details of interfacing with external components.
//
// A nested service is modelled here as a finite state machine, with a Service interface that all nested services
// should implement.
//
// The state machine has the following states:
//   - Ready.  The service is running normally.
//   - Not ready.  The service is temporarily unavailable.
//   - Stopped.  The service is permanently unavailable.
//
// Additionally, an error state is exposed.
//   - When ready, the error state should always be nil.
//   - When not ready, the error state may indicate a reason for being not ready.  Not ready with a nil error state
//     implies that the service is initializing.
//   - When stopped, the error state may indicate a reason for being stopped.  Stopped with a nil error state implies
//     that the service was stopped by the calling process with Stop().
//
// This package also provides a Monitor type, which implements the state machine.  A Monitor can be embedded in
// any service to make it a nested service.
package nested
