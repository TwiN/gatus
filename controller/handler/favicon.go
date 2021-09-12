package handler

import (
	"net/http"
)

// FavIcon handles requests for /favicon.ico
func FavIcon(staticFolder string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, staticFolder+"/favicon.ico")
	}
}
