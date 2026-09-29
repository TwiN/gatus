package client

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/TwiN/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	health "google.golang.org/grpc/health/grpc_health_v1"
)

// PerformGRPCHealthCheck dials a gRPC target and performs the standard Health/Check RPC.
// Returns whether a connection was established, the serving status string, an error (if any), and the elapsed duration.
func PerformGRPCHealthCheck(address string, useTLS bool, cfg *Config) (bool, string, error, time.Duration) {
	if cfg == nil {
		cfg = GetDefaultConfig()
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var opts []grpc.DialOption
	// Transport credentials
	if useTLS {
		tlsCfg := &tls.Config{InsecureSkipVerify: cfg.Insecure}
		if cfg.HasTLSConfig() && cfg.TLS.isValid() == nil {
			tlsCfg = configureTLS(tlsCfg, *cfg.TLS)
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	// Custom dialer for DNS resolver or SSH tunnel
	opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		if cfg.ResolvedTunnel != nil {
			return cfg.ResolvedTunnel.Dial("tcp", addr)
		}
		if cfg.HasCustomDNSResolver() {
			resolverCfg, err := cfg.parseDNSResolver()
			if err != nil {
				// Shouldn't happen because already validated; log and fall back
				logr.Errorf("[client.PerformGRPCHealthCheck] invalid DNS resolver: %v", err)
			} else {
				d := &net.Dialer{Resolver: &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
					d := net.Dialer{}
					return d.DialContext(ctx, resolverCfg.Protocol, resolverCfg.Host+":"+resolverCfg.Port)
				}}}
				return d.DialContext(ctx, "tcp", addr)
			}
		}
		var d net.Dialer
		return d.DialContext(ctx, "tcp", addr)
	}))

	start := time.Now()
	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return false, "", err, time.Since(start)
	}
	defer conn.Close()

	client := health.NewHealthClient(conn)
	resp, err := client.Check(ctx, &health.HealthCheckRequest{Service: ""})
	if err != nil {
		return false, "", err, time.Since(start)
	}
	return true, resp.GetStatus().String(), nil, time.Since(start)
}
