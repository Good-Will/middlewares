package middlewares

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// NewDumpMiddleware creates a new DumpMiddleware to call a function with the RoundtripDump objects.
func NewDumpMiddleware(dumpAction func(*RoundtripDump)) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodOptions {
				sw := NewResponseSnifferingWriter(w)
				// Call the next handler, which can be another middleware in the chain, or the final handler.
				requestData := dumpRequest(r)
				next.ServeHTTP(&sw, r)
				responseData := dumpResponse(&sw)
				dump := RoundtripDump{Timestamp: time.Now(), Request: *requestData, Response: *responseData}
				go dumpAction(&dump)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// NewDumpToLogMiddleware creates a new DumpMiddleware to log the RoundtripDump objects as serialised json string.
func NewDumpToLogMiddleware() func(next http.Handler) http.Handler {
	return NewDumpMiddleware(func(dump *RoundtripDump) {
		marshalledDump, _ := json.Marshal(dump)
		log.Println(string(marshalledDump))
	})
}

// RequestDump  - A RequestDump object represents an HTTP request.
// The HTTP method is stored as a string.
// The HTTP target url is stored as a string.
// The HTTP protocal, e.g. HTTP/HTTPS, is stored as a string.
// The HTTP headers are stored in a string-string map.
// The HTTP body is stored as a string.
type RequestDump struct {
	Method   string              `json:"method"`
	Target   string              `json:"target"`
	Protocol string              `json:"protocol"`
	Headers  map[string][]string `json:"headers"`
	Body     string              `json:"body"`
}

// ResponseDump - A ResponseDump object represents an HTTP response.
// The HTTP headers are stored in a string-string map.
// The HTTP body is stored as a string.
// The HTTP status code is stored as an integer.
type ResponseDump struct {
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code"`
}

// RoundtripDump - A RoundtripDump object represents a full roundtrip of an HTTP call.
type RoundtripDump struct {
	Timestamp time.Time    `json:"timestamp"`
	Request   RequestDump  `json:"request"`
	Response  ResponseDump `json:"response"`
}

func dumpRoundtrip(sw *ResponseSnifferingWriter, r *http.Request) *RoundtripDump {
	requestData := dumpRequest(r)
	responseData := dumpResponse(sw)
	dump := RoundtripDump{Timestamp: time.Now(), Request: *requestData, Response: *responseData}
	return &dump
}

func dumpRequest(r *http.Request) *RequestDump {

	bodyBuf, _ := ioutil.ReadAll(r.Body)

	var bodyString string

	if bodyBuf != nil {
		newBody := ioutil.NopCloser(bytes.NewBuffer(bodyBuf))
		r.Body = newBody
		bodyString = string(bodyBuf)
	}

	rStruct := RequestDump{Headers: make(map[string][]string), Body: bodyString}

	rStruct.Method = r.Method
	rStruct.Target = r.RequestURI
	rStruct.Protocol = r.Proto

	for k, v := range r.Header {
		rStruct.Headers[k] = v
	}

	return &rStruct
}

func dumpResponse(sw *ResponseSnifferingWriter) *ResponseDump {
	headers := sw.ResponseWriter.Header()
	b := sw.BytesBuffer.Bytes()
	// Check that the server actually sent compressed data
	var reader io.Reader = bytes.NewReader(b)

	switch headers.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(reader)
	default:
	}
	b, _ = ioutil.ReadAll(reader)

	rStruct := ResponseDump{Headers: make(map[string]string), Body: string(b), StatusCode: sw.Status}
	for k, v := range headers {
		rStruct.Headers[k] = ""
		for _, vv := range v {
			rStruct.Headers[k] += vv
		}
	}
	return &rStruct
}

// ResponseSnifferingWriter overrides the logic of http.ResponseWriter to dump the full roundtrips of HTTP calls.
type ResponseSnifferingWriter struct {
	http.ResponseWriter
	MultiWriter io.Writer
	BytesBuffer *bytes.Buffer
	Status      int
}

// NewResponseSnifferingWriter initiates a ResponseSnifferingWriter object.
func NewResponseSnifferingWriter(realWriter http.ResponseWriter) ResponseSnifferingWriter {
	result := ResponseSnifferingWriter{ResponseWriter: realWriter}
	result.BytesBuffer = bytes.NewBuffer(nil)
	result.MultiWriter = io.MultiWriter(result.BytesBuffer, realWriter)
	return result
}

// Header overrides the logic of http.ResponseWriter.Header()
func (w *ResponseSnifferingWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// WriteHeader overrides the logic of http.ResponseWriter.WriteHeader()
func (w *ResponseSnifferingWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write overrides the logic of http.ResponseWriter.Write()
func (w *ResponseSnifferingWriter) Write(b []byte) (n int, err error) {
	n, err = w.MultiWriter.Write(b)
	return
}
