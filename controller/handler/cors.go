package handler

import "net/http"

func DevelopmentCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081")
		next.ServeHTTP(w, r)
	})
}
