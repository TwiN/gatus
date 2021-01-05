package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/go-ping/ping"
)

var (
	secureHTTPClient   *http.Client
	insecureHTTPClient *http.Client
)

// GetHTTPClient returns the shared HTTP client
func GetHTTPClient(insecure bool) *http.Client {
	if insecure {
		if insecureHTTPClient == nil {
			insecureHTTPClient = &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 20,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
		return insecureHTTPClient
	}
	if secureHTTPClient == nil {
		secureHTTPClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
			},
		}
	}
	return secureHTTPClient
}

// CanCreateTCPConnection checks whether a connection can be established with a TCP service
func CanCreateTCPConnection(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Ping checks if an address can be pinged and returns the round-trip time if the address can be pinged
//
// Note that this function takes at least 100ms, even if the address is 127.0.0.1
func Ping(address string) (bool, time.Duration) {
	pinger, err := ping.NewPinger(address)
	if err != nil {
		return false, 0
	}
	pinger.Count = 1
	pinger.Timeout = 5 * time.Second
	pinger.SetNetwork("ip4")
	pinger.SetPrivileged(true)
	err = pinger.Run()
	if err != nil {
		return false, 0
	}
	if pinger.Statistics() != nil {
		return true, pinger.Statistics().MaxRtt
	}
	return true, 0
}
