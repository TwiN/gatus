package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"runtime"
	"strings"
	"time"

	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/logr"
	"github.com/TwiN/whois"
	"github.com/ishidawataru/sctp"
	"github.com/miekg/dns"
	ping "github.com/prometheus-community/pro-bing"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
)

const (
	dnsPort = 53
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
	ch := make(chan bool, 1)
	go (func(res chan bool) {
		addr, err := sctp.ResolveSCTPAddr("sctp", address)
		if err != nil {
			res <- false
			return
		}

		conn, err := sctp.DialSCTP("sctp", nil, addr)
		if err != nil {
			res <- false
			return
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

func CheckSSHBanner(address string, cfg *Config) (bool, int, error) {
	var port string
	if strings.Contains(address, ":") {
		addressAndPort := strings.Split(address, ":")
		if len(addressAndPort) != 2 {
			return false, 1, errors.New("invalid address for ssh, format must be ssh://host:port")
		}
		address = addressAndPort[0]
		port = addressAndPort[1]
	} else {
		port = "22"
	}
	dialer := net.Dialer{}
	connStr := net.JoinHostPort(address, port)
	conn, err := dialer.Dial("tcp", connStr)
	if err != nil {
		return false, 1, err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 256)
	_, err = io.ReadAtLeast(conn, buf, 1)
	if err != nil {
		return false, 1, err
	}
	return true, 0, err
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
	var exitErr *ssh.ExitError
	if ok := errors.As(err, &exitErr); !ok {
		return false, 0, err
	}
	return true, exitErr.ExitStatus(), nil
}

// Ping checks if an address can be pinged and returns the round-trip time if the address can be pinged
//
// Note that this function takes at least 100ms, even if the address is 127.0.0.1
func Ping(address string, config *Config) (bool, time.Duration) {
	pinger := ping.New(address)
	pinger.Count = 1
	pinger.Timeout = config.Timeout
	// Set the pinger's privileged mode to true for every GOOS except darwin
	// See https://github.com/TwiN/gatus/issues/132
	//
	// Note that for this to work on Linux, Gatus must run with sudo privileges.
	// See https://github.com/prometheus-community/pro-bing#linux
	pinger.SetPrivileged(runtime.GOOS != "darwin")
	pinger.SetNetwork(config.Network)
	err := pinger.Run()
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

// QueryWebSocket opens a websocket connection, write `body` and return a message from the server
func QueryWebSocket(address, body string, config *Config) (bool, []byte, error) {
	const (
		Origin             = "http://localhost/"
		MaximumMessageSize = 1024 // in bytes
	)
	wsConfig, err := websocket.NewConfig(address, Origin)
	if err != nil {
		return false, nil, fmt.Errorf("error configuring websocket connection: %w", err)
	}
	if config != nil {
		wsConfig.Dialer = &net.Dialer{Timeout: config.Timeout}
	}
	// Dial URL
	ws, err := websocket.DialConfig(wsConfig)
	if err != nil {
		return false, nil, fmt.Errorf("error dialing websocket: %w", err)
	}
	defer ws.Close()
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
	return true, msg[:n], nil
}

func QueryDNS(queryType, queryName, url string) (connected bool, dnsRcode string, body []byte, err error) {
	if !strings.Contains(url, ":") {
		url = fmt.Sprintf("%s:%d", url, dnsPort)
	}
	queryTypeAsUint16 := dns.StringToType[queryType]
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(queryName, queryTypeAsUint16)
	r, _, err := c.Exchange(m, url)
	if err != nil {
		logr.Infof("[client.QueryDNS] Error exchanging DNS message: %v", err)
		return false, "", nil, err
	}
	connected = true
	dnsRcode = dns.RcodeToString[r.Rcode]
	for _, rr := range r.Answer {
		switch rr.Header().Rrtype {
		case dns.TypeA:
			if a, ok := rr.(*dns.A); ok {
				body = []byte(a.A.String())
			}
		case dns.TypeAAAA:
			if aaaa, ok := rr.(*dns.AAAA); ok {
				body = []byte(aaaa.AAAA.String())
			}
		case dns.TypeCNAME:
			if cname, ok := rr.(*dns.CNAME); ok {
				body = []byte(cname.Target)
			}
		case dns.TypeMX:
			if mx, ok := rr.(*dns.MX); ok {
				body = []byte(mx.Mx)
			}
		case dns.TypeNS:
			if ns, ok := rr.(*dns.NS); ok {
				body = []byte(ns.Ns)
			}
		case dns.TypePTR:
			if ptr, ok := rr.(*dns.PTR); ok {
				body = []byte(ptr.Ptr)
			}
		case dns.TypeSRV:
			if srv, ok := rr.(*dns.SRV); ok {
				body = []byte(fmt.Sprintf("%s:%d", srv.Target, srv.Port))
			}
		default:
			body = []byte("query type is not supported yet")
		}
	}
	return connected, dnsRcode, body, nil
}

// InjectHTTPClient is used to inject a custom HTTP client for testing purposes
func InjectHTTPClient(httpClient *http.Client) {
	injectedHTTPClient = httpClient
}
