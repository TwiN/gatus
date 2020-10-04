package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	secureHttpClient   *http.Client
	insecureHttpClient *http.Client
)

func GetHttpClient(insecure bool) *http.Client {
	if insecure {
		if insecureHttpClient == nil {
			insecureHttpClient = &http.Client{
				Timeout: time.Second * 10,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
		return insecureHttpClient
	} else {
		if secureHttpClient == nil {
			secureHttpClient = &http.Client{
				Timeout: time.Second * 10,
			}
		}
		return secureHttpClient
	}
}

func CanCreateConnectionToTcpService(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
