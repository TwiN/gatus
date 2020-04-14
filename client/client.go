package client

import (
	"net/http"
	"time"
)

var (
	client *http.Client
)

func GetHttpClient() *http.Client {
	if client == nil {
		client = &http.Client{
			Timeout: time.Second * 10,
		}
	}
	return client
}
