package tunneling

import (
	"fmt"
	"strings"
	"sync"

	"github.com/TwiN/gatus/v5/config/tunneling/sshtunnel"
)

// Config represents the tunneling configuration
type Config struct {
	// Tunnels is a map of SSH tunnel configurations in which the key is the name of the tunnel
	Tunnels map[string]*sshtunnel.Config `yaml:",inline"`

	mu          sync.RWMutex                    `yaml:"-"`
	connections map[string]*sshtunnel.SSHTunnel `yaml:"-"`
}

// ValidateAndSetDefaults validates the tunneling configuration and sets defaults
func (tc *Config) ValidateAndSetDefaults() error {
	if tc.connections == nil {
		tc.connections = make(map[string]*sshtunnel.SSHTunnel)
	}
	for name, config := range tc.Tunnels {
		if err := config.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("tunnel '%s': %w", name, err)
		}
	}
	return nil
}

// GetTunnel returns the SSH tunnel for the given name, creating it if necessary
func (tc *Config) GetTunnel(name string) (*sshtunnel.SSHTunnel, error) {
	if name == "" {
		return nil, fmt.Errorf("tunnel name cannot be empty")
	}
	tc.mu.Lock()
	defer tc.mu.Unlock()
	// Check if tunnel already exists
	if tunnel, exists := tc.connections[name]; exists {
		return tunnel, nil
	}
	// Get config for this tunnel
	config, exists := tc.Tunnels[name]
	if !exists {
		return nil, fmt.Errorf("tunnel '%s' not found in configuration", name)
	}
	// Create and store new tunnel
	tunnel := sshtunnel.New(config)
	tc.connections[name] = tunnel
	return tunnel, nil
}

// Close closes all SSH tunnel connections
func (tc *Config) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	var errors []string
	for name, tunnel := range tc.connections {
		if err := tunnel.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("tunnel '%s': %v", name, err))
		}
		delete(tc.connections, name)
	}
	if len(errors) > 0 {
		return fmt.Errorf("failed to close tunnels: %s", strings.Join(errors, ", "))
	}
	return nil
}
