package watchdog

import "sync"

var endpointLocks sync.Map

// lockEndpoint locks the mutex of the given name, creating it if it does not exist yet, and returns the function that unlocks it.
func lockEndpoint(name string) (unlock func()) {
	mutex, _ := endpointLocks.LoadOrStore(name, &sync.Mutex{})
	mutex.(*sync.Mutex).Lock()
	return mutex.(*sync.Mutex).Unlock
}

// LockExternalEndpoint serializes the handling of the results of an external endpoint.
//
// An external endpoint is handled from two goroutines, the one that serves the push request and the one that checks the
// heartbeat. Both copy the endpoint, evaluate the alerts, which are shared between the copies, and write the counters back.
// Without this lock they race on the alerts and on the counters, and can both trigger the same alert.
func LockExternalEndpoint(key string) (unlock func()) {
	return lockEndpoint("external:" + key)
}
