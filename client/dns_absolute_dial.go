package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// newAbsoluteLookupDialer returns a dialer whose hostname resolution goes
// through the configured custom DNS resolver with FQDN semantics: the
// hostname is looked up with a trailing dot, which the pure-Go resolver
// treats as fully qualified and therefore does NOT expand through the
// system search list from /etc/resolv.conf (#1769).
//
// Why the trailing dot matters: a `PreferGo` resolver still reads
// /etc/resolv.conf for `search`/`ndots`, and on Kubernetes every pod gets
// `options ndots:5` — a normal monitored hostname has fewer than 5 dots, is
// treated as relative, and the whole search list is walked first
// (namespace.svc.cluster.local, svc.cluster.local, cluster.local, ...) with
// the intended absolute query only last. Doubled by A and AAAA lookups, that
// is 8 queries per check — 6 guaranteed NXDOMAINs — all sent to the
// configured public resolver, disclosing the cluster's namespace layout.
//
// The endpoint hostname in a URL is always meant as an absolute name, so
// none of that expansion is wanted. Once the IPs are resolved absolutely,
// the dial itself connects to the IP, bypassing the resolver entirely.
func newAbsoluteLookupDialer(dnsResolver *DNSResolverConfig) *net.Dialer {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, dnsResolver.Protocol, dnsResolver.Host+":"+dnsResolver.Port)
		},
	}
	return &net.Dialer{
		Timeout:  30 * time.Second,
		Resolver: resolver,
	}
}

// absoluteLookupDialContext resolves host with FQDN semantics through the
// custom resolver and dials the first IP. addr is "host:port".
func absoluteLookupDialContext(ctx context.Context, dialer *net.Dialer, dnsResolver *DNSResolverConfig, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("client.absoluteLookupDialContext: %w", err)
	}
	if ip := net.ParseIP(host); ip != nil {
		// Literal addresses never hit the search list; dial directly.
		return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
	}
	// The trailing dot is the FQDN marker that disables search-list
	// expansion in the pure-Go resolver.
	fqdn := host
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}
	ips, err := dialer.Resolver.LookupHost(ctx, fqdn)
	if err != nil {
		return nil, fmt.Errorf("client.absoluteLookupDialContext: resolving %s via %s:%s: %w", host, dnsResolver.Host, dnsResolver.Port, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("client.absoluteLookupDialContext: %s resolved to no addresses", host)
	}
	var conn net.Conn
	var lastErr error
	for _, ip := range ips {
		conn, lastErr = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if lastErr == nil {
			return conn, nil
		}
	}
	return nil, lastErr
}
