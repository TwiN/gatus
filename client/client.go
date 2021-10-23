package client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/smtp"
	"runtime"
	"strings"
	"time"

	"github.com/go-ping/ping"
)

// GetHTTPClient returns the shared HTTP client
func GetHTTPClient(config *Config) *http.Client {
	if config == nil {
		return defaultConfig.getHTTPClient()
	}
	return config.getHTTPClient()
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
