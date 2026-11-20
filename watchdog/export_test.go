package watchdog

import (
	"context"

	"golang.org/x/sync/semaphore"
)

// ExecuteExternalEndpointHeartbeat and InitializeForTest let the tests of package watchdog_test drive the heartbeat the
// way the monitoring loop does.
var ExecuteExternalEndpointHeartbeat = executeExternalEndpointHeartbeat

func InitializeForTest() {
	ctx, monitoringSemaphore = context.Background(), semaphore.NewWeighted(UnlimitedConcurrencyWeight)
}
