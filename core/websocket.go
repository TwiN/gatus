package core

import (
	"golang.org/x/net/websocket"
)

func queryWebSocket(endpoint *Endpoint, result *Result) {
	const (
		Origin             = "http://localhost/"
		MaximumMessageSize = 1024 // in bytes
	)

	url := endpoint.URL
	message := endpoint.Body

	config, err := websocket.NewConfig(url, Origin)
	if err != nil {
		result.AddError("Error configuring WS connection:" + err.Error())
		return
	}

	// Dial URL
	ws, err := websocket.DialConfig(config)
	if err != nil {
		result.AddError("Error dialing WS:" + err.Error())
		return
	}
	result.Connected = true

	// Write message
	if _, err := ws.Write([]byte(message)); err != nil {
		result.AddError("Error writing WS message" + err.Error())
		return
	}

	// Read message
	var msg = make([]byte, MaximumMessageSize)
	var n int
	if n, err = ws.Read(msg); err != nil {
		result.AddError("Error reading WS message" + err.Error())
		return
	}

	result.Body = msg[:n]

	// Close socket
	defer ws.Close()
}
