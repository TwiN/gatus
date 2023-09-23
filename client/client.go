package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/websocket"
	"net"
	"net/http"
	"net/smtp"
	"runtime"
	"strings"
	"time"

	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/whois"
	"github.com/ishidawataru/sctp"
	ping "github.com/prometheus-community/pro-bing"
	"golang.org/x/crypto/ssh"
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
	connection, err := tls.DialWithDialer(&net.Dialer{Timeout: config.Timeout}, "tcp", address, &tls.Config{
		InsecureSkipVerify: config.Insecure,
	})
	if err != nil {
		return
	}
	defer connection.Close()
	verifiedChains := connection.ConnectionState().VerifiedChains
	// If config.Insecure is set to true, verifiedChains will be an empty list []
	// We should get the parsed certificates from PeerCertificates, it can't be empty on the client side
	// Reference: https://pkg.go.dev/crypto/tls#PeerCertificates
	if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
		peerCertificates := connection.ConnectionState().PeerCertificates
		return true, peerCertificates[0], nil
	}
	return true, verifiedChains[0][0], nil
}

// CanCreateSSHConnection checks whether a connection can be established and a command can be executed to an address
// using the SSH protocol.
func CanCreateSSHConnection(address, username, password string, config *Config) (bool, *ssh.Client, error) {
	var port string
	if strings.Contains(address, ":") {
		addressAndPort := strings.Split(address, ":")
		if len(addressAndPort) != 2 {
			return false, nil, errors.New("invalid address for ssh, format must be host:port")
		}
		address = addressAndPort[0]
		port = addressAndPort[1]
	} else {
		port = "22"
	}

	cli, err := ssh.Dial("tcp", strings.Join([]string{address, port}, ":"), &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout: config.Timeout,
	})
	if err != nil {
		return false, nil, err
	}

	return true, cli, nil
}

// ExecuteSSHCommand executes a command to an address using the SSH protocol.
func ExecuteSSHCommand(sshClient *ssh.Client, body string, config *Config) (bool, int, error) {
	type Body struct {
		Command string `json:"command"`
	}

	defer sshClient.Close()

	var b Body
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		return false, 0, err
	}

	sess, err := sshClient.NewSession()
	if err != nil {
		return false, 0, err
	}

	err = sess.Start(b.Command)
	if err != nil {
		return false, 0, err
	}

	defer sess.Close()

	err = sess.Wait()
	if err == nil {
		return true, 0, nil
	}

	e, ok := err.(*ssh.ExitError)
	if !ok {
		return false, 0, err
	}

	return true, e.ExitStatus(), nil
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
	// See https://github.com/prometheus-community/pro-bing#linux
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

// Open a websocket connection, write `body` and return a message from the server
func QueryWebSocket(address string, config *Config, body string) (bool, []byte, error) {
	const (
		Origin             = "http://localhost/"
		MaximumMessageSize = 1024 // in bytes
	)

	wsConfig, err := websocket.NewConfig(address, Origin)
	if err != nil {
		return false, nil, fmt.Errorf("error configuring websocket connection: %w", err)
	}
	// Dial URL
	ws, err := websocket.DialConfig(wsConfig)
	if err != nil {
		return false, nil, fmt.Errorf("error dialing websocket: %w", err)
	}
	defer ws.Close()
	connected := true
	// Write message
	if _, err := ws.Write([]byte(body)); err != nil {
		return false, nil, fmt.Errorf("error writing websocket body: %w", err)
	}
	// Read message
	var n int
	msg := make([]byte, MaximumMessageSize)
	if n, err = ws.Read(msg); err != nil {
		return false, nil, fmt.Errorf("error reading websocket message: %w", err)
	}
	return connected, msg[:n], nil
}

// InjectHTTPClient is used to inject a custom HTTP client for testing purposes
func InjectHTTPClient(httpClient *http.Client) {
	injectedHTTPClient = httpClient
}
