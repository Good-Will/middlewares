package middlewares

import (
	"net/http"
)

// NewRequestHeaderWriteMiddlwware creates a middleware to rewrite HTTP headers of requests.
func NewRequestHeaderWriteMiddlwware(headers map[string]string) func(next http.Handler) http.Handler {
	rewriteHeader := castToHeader(headers)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range rewriteHeader {
				r.Header[k] = v
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewResponseHeaderWriteMiddlwware creates a middleware to rewrite HTTP headers of responses.
func NewResponseHeaderWriteMiddlwware(headers map[string]string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rewriteHeader := castToHeaderForRequest(headers, r)
			rw := rewriteResponseWriter{ResponseWriter: w, RewriteHeader: rewriteHeader}
			next.ServeHTTP(rw, r)
		})
	}
}

func castToHeader(c map[string]string) http.Header {
	rewriteHeader := make(http.Header)
	for k, v := range c {
		rewriteHeader[k] = []string{v}
	}
	return rewriteHeader
}

func castToHeaderForRequest(c map[string]string, r *http.Request) http.Header {
	rewriteHeader := make(http.Header)
	for k, v := range c {
		if k == "Access-Control-Allow-Origin" && v == "*" {
			rewriteHeader[k] = r.Header["Origin"]
		} else {
			rewriteHeader[k] = []string{v}
		}
	}
	return rewriteHeader
}

// rewriteResponseWriter overrides the logic of http.ResponseWriter to rewrite the HTTP headers of requests or responses.
type rewriteResponseWriter struct {
	http.ResponseWriter
	RewriteHeader http.Header
}

// Header overrides the logic of http.ResponseWriter.Header()
func (w rewriteResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write overrides the logic of http.ResponseWriter.Write()
func (w rewriteResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// WriteHeader overrides the logic of http.ResponseWriter.WriteHeader
func (w rewriteResponseWriter) WriteHeader(statusCode int) {

	for k, v := range w.RewriteHeader {
		w.Header()[k] = v
	}

	if len(w.Header()["Access-Control-Allow-Origin"]) > 0 && len(w.Header()["Access-Control-Allow-Headers"]) == 0 {
		w.Header()["Access-Control-Allow-Headers"] = []string{"*"}
	}

	w.ResponseWriter.WriteHeader(statusCode)
}
