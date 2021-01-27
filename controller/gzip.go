package controller

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

var gzPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(ioutil.Discard)
	},
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// WriteHeader sends an HTTP response header with the provided status code.
// It also deletes the Content-Length header, since the GZIP compression may modify the size of the payload
func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

// Write writes len(b) bytes from b to the underlying data stream.
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipHandler compresses the response of a given http.Handler if the request's headers specify that the client
// supports gzip encoding
func GzipHandler(next http.Handler) http.Handler {
	return GzipHandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(writer, r)
	})
}

// GzipHandlerFunc compresses the response of a given http.HandlerFunc if the request's headers specify that the client
// supports gzip encoding
func GzipHandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, r *http.Request) {
		// If the request doesn't specify that it supports gzip, then don't compress it
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(writer, r)
			return
		}
		writer.Header().Set("Content-Encoding", "gzip")
		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)
		gz.Reset(writer)
		defer gz.Close()
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: writer, Writer: gz}, r)
	}
}
