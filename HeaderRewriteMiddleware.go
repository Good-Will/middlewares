package middlewares

import (
	"net/http"
)

// NewRequestHeaderWriteMiddlwware TODO
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

// NewResponseHeaderWriteMiddlwware TODO
func NewResponseHeaderWriteMiddlwware(headers map[string]string) func(next http.Handler) http.Handler {
	rewriteHeader := castToHeader(headers)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// ResponseWriter TODO
type rewriteResponseWriter struct {
	http.ResponseWriter
	RewriteHeader http.Header
}

// Header TODO
func (w rewriteResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write TODO
func (w rewriteResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// WriteHeader TODO
func (w rewriteResponseWriter) WriteHeader(statusCode int) {

	for k, v := range w.RewriteHeader {
		w.Header()[k] = v
	}

	if len(w.Header()["Access-Control-Allow-Origin"]) > 0 && len(w.Header()["Access-Control-Allow-Headers"]) == 0 {
		w.Header()["Access-Control-Allow-Headers"] = []string{"*"}
	}

	w.ResponseWriter.WriteHeader(statusCode)
}
