package core

import (
	"golang.org/x/net/websocket"
)

func queryWebSocket(endpoint *Endpoint, result *Result) {
	origin := "http://localhost/"
	url := endpoint.URL
	message := endpoint.Body

	config, err := websocket.NewConfig(url, origin)
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

	// Read message (1 kByte)
	var msg = make([]byte, 1024)
	var n int
	if n, err = ws.Read(msg); err != nil {
		result.AddError("Error reading WS message" + err.Error())
		return
	}

	result.Body = msg[:n]
}
