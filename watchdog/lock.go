package watchdog

import "github.com/TwiN/gatus/v5/config/endpoint"

// LockExternalEndpoint serializes the handling of the results of an external endpoint, from the push handler and from the heartbeat.
func LockExternalEndpoint(ee *endpoint.ExternalEndpoint) (unlock func()) {
	return ee.Lock()
}
