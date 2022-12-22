package client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"runtime"
	"strings"
	"time"

	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/whois"
	"github.com/go-ping/ping"
	"github.com/ishidawataru/sctp"
)

var (
	// injectedHTTPClient is used for testing purposes
	injectedHTTPClient *http.Client

	whoisClient              = whois.NewClient().WithReferralCache(true)
	whoisExpirationDateCache = gocache.NewCache().WithMaxSize(10000).WithDefaultTTL(24 * time.Hour)
)

// GetHTTPClient returns the shared HTTP client, or the client from the configuration passed
func GetHTTPClient(config *Config) *http.Client {
	if injectedHTTPClient != nil {
		return injectedHTTPClient
	}
	if config == nil {
		return defaultConfig.getHTTPClient()
	}
	return config.getHTTPClient()
}

// GetDomainExpiration retrieves the duration until the domain provided expires
func GetDomainExpiration(hostname string) (domainExpiration time.Duration, err error) {
	var retrievedCachedValue bool
	if v, exists := whoisExpirationDateCache.Get(hostname); exists {
		domainExpiration = time.Until(v.(time.Time))
		retrievedCachedValue = true
		// If the domain OR the TTL is not going to expire in less than 24 hours
		// we don't have to refresh the cache. Otherwise, we'll refresh it.
		cacheEntryTTL, _ := whoisExpirationDateCache.TTL(hostname)
		if cacheEntryTTL > 24*time.Hour && domainExpiration > 24*time.Hour {
			// No need to refresh, so we'll just return the cached values
			return domainExpiration, nil
		}
	}
	if whoisResponse, err := whoisClient.QueryAndParse(hostname); err != nil {
		if !retrievedCachedValue { // Add an error unless we already retrieved a cached value
			return 0, fmt.Errorf("error querying and parsing hostname using whois client: %w", err)
		}
	} else {
		domainExpiration = time.Until(whoisResponse.ExpirationDate)
		if domainExpiration > 720*time.Hour {
			whoisExpirationDateCache.SetWithTTL(hostname, whoisResponse.ExpirationDate, 240*time.Hour)
		} else {
			whoisExpirationDateCache.SetWithTTL(hostname, whoisResponse.ExpirationDate, 72*time.Hour)
		}
	}
	return domainExpiration, nil
}

// CanCreateTCPConnection checks whether a connection can be established with a TCP endpoint
func CanCreateTCPConnection(address string, config *Config) bool {
	conn, err := net.DialTimeout("tcp", address, config.Timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// CanCreateUDPConnection checks whether a connection can be established with a UDP endpoint
func CanCreateUDPConnection(address string, config *Config) bool {
	conn, err := net.DialTimeout("udp", address, config.Timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// CanCreateSCTPConnection checks whether a connection can be established with a SCTP endpoint
func CanCreateSCTPConnection(address string, config *Config) bool {
	ch := make(chan bool)
	go (func(res chan bool) {
		addr, err := sctp.ResolveSCTPAddr("sctp", address)
		if err != nil {
			res <- false
		}

		conn, err := sctp.DialSCTP("sctp", nil, addr)
		if err != nil {
			res <- false
		}
		_ = conn.Close()
		res <- true
	})(ch)

	select {
	case result := <-ch:
		return result
	case <-time.After(config.Timeout):
		return false
	}
}

// CanPerformStartTLS checks whether a connection can be established to an address using the STARTTLS protocol
func CanPerformStartTLS(address string, config *Config) (connected bool, certificate *x509.Certificate, err error) {
	hostAndPort := strings.Split(address, ":")
	if len(hostAndPort) != 2 {
		return false, nil, errors.New("invalid address for starttls, format must be host:port")
	}
	connection, err := net.DialTimeout("tcp", address, config.Timeout)
	if err != nil {
		return
	}
	smtpClient, err := smtp.NewClient(connection, hostAndPort[0])
	if err != nil {
		return
	}
	err = smtpClient.StartTLS(&tls.Config{
		InsecureSkipVerify: config.Insecure,
		ServerName:         hostAndPort[0],
	})
	if err != nil {
		return
	}
	if state, ok := smtpClient.TLSConnectionState(); ok {
		certificate = state.PeerCertificates[0]
	} else {
		return false, nil, errors.New("could not get TLS connection state")
	}
	return true, certificate, nil
}

// CanPerformTLS checks whether a connection can be established to an address using the TLS protocol
func CanPerformTLS(address string, config *Config) (connected bool, certificate *x509.Certificate, err error) {
	connection, err := tls.DialWithDialer(&net.Dialer{Timeout: config.Timeout}, "tcp", address, nil)
	if err != nil {
		return
	}
	defer connection.Close()
	verifiedChains := connection.ConnectionState().VerifiedChains
	if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
		return
	}
	return true, verifiedChains[0][0], nil
}

// Ping checks if an address can be pinged and returns the round-trip time if the address can be pinged
//
// Note that this function takes at least 100ms, even if the address is 127.0.0.1
func Ping(address string, config *Config) (bool, time.Duration) {
	pinger, err := ping.NewPinger(address)
	if err != nil {
		return false, 0
	}
	pinger.Count = 1
	pinger.Timeout = config.Timeout
	// Set the pinger's privileged mode to true for every GOOS except darwin
	// See https://github.com/TwiN/gatus/issues/132
	//
	// Note that for this to work on Linux, Gatus must run with sudo privileges.
	// See https://github.com/go-ping/ping#linux
	pinger.SetPrivileged(runtime.GOOS != "darwin")
	err = pinger.Run()
	if err != nil {
		return false, 0
	}
	if pinger.Statistics() != nil {
		// If the packet loss is 100, it means that the packet didn't reach the host
		if pinger.Statistics().PacketLoss == 100 {
			return false, pinger.Timeout
		}
		return true, pinger.Statistics().MaxRtt
	}
	return true, 0
}

// InjectHTTPClient is used to inject a custom HTTP client for testing purposes
func InjectHTTPClient(httpClient *http.Client) {
	injectedHTTPClient = httpClient
}
