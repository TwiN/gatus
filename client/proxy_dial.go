package client

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// dialThroughProxy establishes a TCP connection to address by tunneling
// through the proxy named by config.ProxyURL. HTTP and HTTPS proxies are
// reached with a CONNECT handshake; socks5 proxies are dialed directly
// (hostnames are resolved by the proxy, which is the point of routing a
// connectivity check through it). Any other scheme falls back to a plain
// dial, matching how little the raw-dial path can assume about its callers.
func dialThroughProxy(config *Config, address string) (net.Conn, error) {
	proxyURL, err := url.Parse(config.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("client.DialThroughProxy: %w", err)
	}
	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https":
		return dialThroughHTTPConnectProxy(proxyURL, address, config.Timeout)
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if proxyURL.User != nil {
			password, _ := proxyURL.User.Password()
			auth = &proxy.Auth{User: proxyURL.User.Username(), Password: password}
		}
		dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, &net.Dialer{Timeout: config.Timeout})
		if err != nil {
			return nil, fmt.Errorf("client.DialThroughProxy: %w", err)
		}
		return dialer.Dial("tcp", address)
	default:
		return net.DialTimeout("tcp", address, config.Timeout)
	}
}

// dialThroughHTTPConnectProxy performs the RFC 7231 CONNECT handshake against
// proxy and returns the tunneled connection.
func dialThroughHTTPConnectProxy(proxyURL *url.URL, address string, timeout time.Duration) (net.Conn, error) {
	proxyConn, err := net.DialTimeout("tcp", proxyURL.Host, timeout)
	if err != nil {
		return nil, fmt.Errorf("client.DialThroughProxy: %w", err)
	}
	if err := proxyConn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = proxyConn.Close()
		return nil, fmt.Errorf("client.DialThroughProxy: %w", err)
	}
	request := "CONNECT " + address + " HTTP/1.1\r\nHost: " + address + "\r\n"
	if proxyURL.User != nil {
		// RFC 7617 basic proxy authentication.
		password, _ := proxyURL.User.Password()
		credentials := base64.StdEncoding.EncodeToString([]byte(proxyURL.User.Username() + ":" + password))
		request += "Proxy-Authorization: Basic " + credentials + "\r\n"
	}
	request += "\r\n"
	if _, err := proxyConn.Write([]byte(request)); err != nil {
		_ = proxyConn.Close()
		return nil, fmt.Errorf("client.DialThroughProxy: %w", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(proxyConn), nil)
	if err != nil {
		_ = proxyConn.Close()
		return nil, fmt.Errorf("client.DialThroughProxy: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_ = proxyConn.Close()
		return nil, fmt.Errorf("client.DialThroughProxy: proxy answered CONNECT with %s", response.Status)
	}
	_ = proxyConn.SetDeadline(time.Time{})
	return proxyConn, nil
}
