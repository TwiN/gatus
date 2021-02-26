package controller

import "net/http"

// spaHandler handles requests for /
func spaHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, staticFolder+"/index.html")
}
