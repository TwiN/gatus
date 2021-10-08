package health

type Status string

var (
	Down Status = "DOWN" // For when the application is unhealthy
	Up   Status = "UP"   // For when the application is healthy
)
