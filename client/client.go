package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/logr"
	"github.com/TwiN/whois"
	"github.com/gorilla/websocket"
	"github.com/ishidawataru/sctp"
	"github.com/miekg/dns"
	ping "github.com/prometheus-community/pro-bing"
	"github.com/registrobr/rdap"
	"github.com/registrobr/rdap/protocol"
	"golang.org/x/crypto/ssh"
)

const (
	dnsPort = 53
)

var (
	// injectedHTTPClient is used for testing purposes
	injectedHTTPClient *http.Client

	whoisClient              = whois.NewClient().WithReferralCache(true)
	whoisExpirationDateCache = gocache.NewCache().WithMaxSize(10000).WithDefaultTTL(24 * time.Hour)
	rdapClient               = rdap.NewClient(nil)
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
	whoisResponse, err := rdapQuery(hostname)
	if err != nil {
		// fallback to WHOIS protocol
		whoisResponse, err = whoisClient.QueryAndParse(hostname)
	}
	if err != nil {
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

// parseLocalAddressPlaceholder returns a string with the local address replaced
func parseLocalAddressPlaceholder(item string, localAddr net.Addr) string {
	item = strings.ReplaceAll(item, "[LOCAL_ADDRESS]", localAddr.String())
	return item
}

// CanCreateNetworkConnection checks whether a connection can be established with a TCP or UDP endpoint
func CanCreateNetworkConnection(netType string, address string, body string, config *Config) (bool, []byte) {
	const (
		MaximumMessageSize = 1024 // in bytes
	)
	connection, err := net.DialTimeout(netType, address, config.Timeout)
	if err != nil {
		return false, nil
	}
	defer connection.Close()
	if body != "" {
		body = parseLocalAddressPlaceholder(body, connection.LocalAddr())
		connection.SetDeadline(time.Now().Add(config.Timeout))
		_, err = connection.Write([]byte(body))
		if err != nil {
			return false, nil
		}
		buf := make([]byte, MaximumMessageSize)
		n, err := connection.Read(buf)
		if err != nil {
			return false, nil
		}
		return true, buf[:n]
	}
	return true, nil
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

	var connection net.Conn
	var dnsResolver *DNSResolverConfig

	if config.HasCustomDNSResolver() {
		dnsResolver, err = config.parseDNSResolver()

		if err != nil {
			// We're ignoring the error, because it should have been validated on startup ValidateAndSetDefaults.
			// It shouldn't happen, but if it does, we'll log it... Better safe than sorry ;)
			logr.Errorf("[client.getHTTPClient] THIS SHOULD NOT HAPPEN. Silently ignoring invalid DNS resolver due to error: %s", err.Error())
		} else {
			dialer := &net.Dialer{
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						d := net.Dialer{}
						return d.DialContext(ctx, dnsResolver.Protocol, dnsResolver.Host+":"+dnsResolver.Port)
					},
				},
			}
			connection, err = dialer.DialContext(context.Background(), "tcp", address)
			if err != nil {
				return
			}
		}
	} else {
		connection, err = net.DialTimeout("tcp", address, config.Timeout)
		if err != nil {
			return
		}
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
func CanPerformTLS(address string, body string, config *Config) (connected bool, response []byte, certificate *x509.Certificate, err error) {
	const (
		MaximumMessageSize = 1024 // in bytes
	)
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
		certificate = peerCertificates[0]
	} else {
		certificate = verifiedChains[0][0]
	}
	connected = true
	if body != "" {
		body = parseLocalAddressPlaceholder(body, connection.LocalAddr())
		connection.SetDeadline(time.Now().Add(config.Timeout))
		_, err = connection.Write([]byte(body))
		if err != nil {
			return
		}
		buf := make([]byte, MaximumMessageSize)
		var n int
		n, err = connection.Read(buf)
		if err != nil {
			return
		}
		response = buf[:n]
	}
	return
}

// CanCreateSSHConnection checks whether a connection can be established and a command can be executed to an address
// using the SSH protocol.
func CanCreateSSHConnection(address, username, password, privateKey string, config *Config) (bool, *ssh.Client, error) {
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

	// Build auth methods: prefer parsed private key if provided, fall back to password.
	var authMethods []ssh.AuthMethod
	if len(privateKey) > 0 {
		if signer, err := ssh.ParsePrivateKey([]byte(privateKey)); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		} else {
			return false, nil, fmt.Errorf("invalid private key: %w", err)
		}
	}
	if len(password) > 0 {
		authMethods = append(authMethods, ssh.Password(password))
	}

	cli, err := ssh.Dial("tcp", net.JoinHostPort(address, port), &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            username,
		Auth:            authMethods,
		Timeout:         config.Timeout,
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
func ExecuteSSHCommand(sshClient *ssh.Client, body string, config *Config) (bool, int, []byte, error) {
	type Body struct {
		Command string `json:"command"`
	}
	defer sshClient.Close()
	var b Body
	body = parseLocalAddressPlaceholder(body, sshClient.Conn.LocalAddr())
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		return false, 0, nil, err
	}
	sess, err := sshClient.NewSession()
	if err != nil {
		return false, 0, nil, err
	}
	// Capture stdout
	var stdout bytes.Buffer
	sess.Stdout = &stdout
	err = sess.Start(b.Command)
	if err != nil {
		return false, 0, nil, err
	}
	defer sess.Close()
	err = sess.Wait()
	output := stdout.Bytes()
	if err == nil {
		return true, 0, output, nil
	}
	var exitErr *ssh.ExitError
	if ok := errors.As(err, &exitErr); !ok {
		return false, 0, nil, err
	}
	return true, exitErr.ExitStatus(), output, nil
}

// Ping checks if an address can be pinged and returns the round-trip time if the address can be pinged
//
// Note that this function takes at least 100ms, even if the address is 127.0.0.1
func Ping(address string, config *Config) (bool, time.Duration) {
	pinger := ping.New(address)
	pinger.Count = 1
	pinger.Timeout = config.Timeout
	pinger.SetPrivileged(ShouldRunPingerAsPrivileged())
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

// ShouldRunPingerAsPrivileged will determine whether or not to run pinger in privileged mode.
// It should be set to privileged when running as root, and always on windows. See https://pkg.go.dev/github.com/macrat/go-parallel-pinger#Pinger.SetPrivileged
func ShouldRunPingerAsPrivileged() bool {
	// Set the pinger's privileged mode to false for darwin
	// See https://github.com/TwiN/gatus/issues/132
	// linux should also be set to false, but there are potential complications
	// See https://github.com/TwiN/gatus/pull/748 and https://github.com/TwiN/gatus/issues/697#issuecomment-2081700989
	//
	// Note that for this to work on Linux, Gatus must run with sudo privileges. (in certain cases)
	// See https://github.com/prometheus-community/pro-bing#linux
	if runtime.GOOS == "windows" {
		return true
	}
	// To actually check for cap_net_raw capabilities, we would need to add "kernel.org/pub/linux/libs/security/libcap/cap" to gatus.
	// Or use a syscall and check for permission errors, but this requires platform specific compilation
	// As a backstop we can simply check the effective user id and run as privileged when running as root
	return os.Geteuid() == 0
}

// QueryWebSocket opens a websocket connection, write `body` and return a message from the server
func QueryWebSocket(address, body string, headers map[string]string, config *Config) (bool, []byte, error) {
	const (
		Origin = "http://localhost/"
	)
	var (
		dialer = websocket.Dialer{
			EnableCompression: true,
		}
		wsHeaders = make(http.Header)
	)

	wsHeaders.Set("Origin", Origin)
	for name, value := range headers {
		wsHeaders.Set(name, value)
	}

	ctx := context.Background()
	if config != nil {
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: config.Insecure,
		}
		if config.HasTLSConfig() && config.TLS.isValid() == nil {
			dialer.TLSClientConfig = configureTLS(dialer.TLSClientConfig, *config.TLS)
		}
	}
	// Dial URL
	ws, _, err := dialer.DialContext(ctx, address, wsHeaders)
	if err != nil {
		return false, nil, fmt.Errorf("error dialing websocket: %w", err)
	}
	defer ws.Close()
	body = parseLocalAddressPlaceholder(body, ws.LocalAddr())
	// Write message
	if err := ws.WriteMessage(websocket.TextMessage, []byte(body)); err != nil {
		return false, nil, fmt.Errorf("error writing websocket body: %w", err)
	}
	// Read message
	msgType, msg, err := ws.ReadMessage()
	if err != nil {
		return false, nil, fmt.Errorf("error reading websocket message: %w", err)
	} else if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
		return false, nil, fmt.Errorf("unexpected websocket message type: %d, expected %d or %d", msgType, websocket.TextMessage, websocket.BinaryMessage)
	}
	return true, msg, nil
}

func QueryDNS(queryType, queryName, url string) (connected bool, dnsRcode string, body []byte, err error) {
	if !strings.Contains(url, ":") {
		url = fmt.Sprintf("%s:%d", url, dnsPort)
	}
	queryTypeAsUint16 := dns.StringToType[queryType]
	// Special handling: if this is a PTR query and queryName looks like a plain IP,
	// convert it to the proper reverse lookup domain automatically.
	if queryTypeAsUint16 == dns.TypePTR &&
		!strings.HasSuffix(queryName, ".in-addr.arpa.") &&
		!strings.HasSuffix(queryName, ".ip6.arpa.") {
		if rev, convErr := reverseNameForIP(queryName); convErr == nil {
			queryName = rev
		} else {
			return false, "", nil, convErr
		}
	}
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

// rdapQuery returns domain expiration via RDAP protocol
func rdapQuery(hostname string) (*whois.Response, error) {
	data, _, err := rdapClient.Query(hostname, nil, nil)
	if err != nil {
		return nil, err
	}
	domain, ok := data.(*protocol.Domain)
	if !ok {
		return nil, errors.New("invalid domain type")
	}
	response := whois.Response{}
	for _, e := range domain.Events {
		if e.Action == "expiration" {
			response.ExpirationDate = e.Date.Time
			break
		}
	}
	return &response, nil
}

// helper to reverse IP and add in-addr.arpa. IPv4 and IPv6
func reverseNameForIP(ipStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP: %s", ipStr)
	}

	if ipv4 := ip.To4(); ipv4 != nil {
		parts := strings.Split(ipv4.String(), ".")
		for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
			parts[i], parts[j] = parts[j], parts[i]
		}
		return strings.Join(parts, ".") + ".in-addr.arpa.", nil
	}

	ip = ip.To16()
	hexStr := hex.EncodeToString(ip)
	nibbles := strings.Split(hexStr, "")
	for i, j := 0, len(nibbles)-1; i < j; i, j = i+1, j-1 {
		nibbles[i], nibbles[j] = nibbles[j], nibbles[i]
	}
	return strings.Join(nibbles, ".") + ".ip6.arpa.", nil
}
