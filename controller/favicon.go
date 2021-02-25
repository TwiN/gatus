package controller

import "net/http"

// favIconHandler handles requests for /favicon.ico
func favIconHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, staticFolder+"/favicon.ico")
}
