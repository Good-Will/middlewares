package middlewares

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

// NewDumpMiddleware TODO
func NewDumpMiddleware(dumpAction func(*RoundtripDump)) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodOptions {
				sw := NewResponseSnifferingWriter(w)
				// Call the next handler, which can be another middleware in the chain, or the final handler.
				next.ServeHTTP(&sw, r)
				dump := dumpRoundtrip(&sw, r)
				go dumpAction(dump)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// NewDumpToLogMiddleware TODO
func NewDumpToLogMiddleware() func(next http.Handler) http.Handler {
	return NewDumpMiddleware(func(dump *RoundtripDump) {
		marshalledDump, _ := json.Marshal(dump)
		log.Println(string(marshalledDump))
	})
}

// RequestDump  TODO
type RequestDump struct {
	Method   string            `json:"method"`
	Target   string            `json:"target"`
	Protocol string            `json:"protocol"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
}

// ResponseDump TODO
type ResponseDump struct {
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code"`
}

// RoundtripDump TODO
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

	reqBuf, _ := httputil.DumpRequestOut(r, true)

	reqStr := string(reqBuf)
	reqLines := strings.Split(reqStr, "\r\n")

	rStruct := RequestDump{Headers: make(map[string]string), Body: ""}

	inBody := false
	for lineNo, line := range reqLines {
		if lineNo == 0 {
			lineSplit := strings.Split(line, " ")
			if len(lineSplit) > 2 {
				rStruct.Method = lineSplit[0]
				rStruct.Target = lineSplit[1]
				rStruct.Protocol = lineSplit[2]
			}
		} else if !inBody {
			lineSplit := strings.Split(line, ": ")
			if len(lineSplit) >= 2 {
				rStruct.Headers[lineSplit[0]] = lineSplit[1]
			} else {
				inBody = true
			}
		} else {
			if lineNo+1 >= len(reqLines) {
				rStruct.Body += line
			} else {
				rStruct.Body += line + "\r\n"
			}
		}
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

// ResponseSnifferingWriter TODO
type ResponseSnifferingWriter struct {
	http.ResponseWriter
	MultiWriter io.Writer
	BytesBuffer *bytes.Buffer
	Status      int
}

// NewResponseSnifferingWriter TODO
func NewResponseSnifferingWriter(realWriter http.ResponseWriter) ResponseSnifferingWriter {
	result := ResponseSnifferingWriter{ResponseWriter: realWriter}
	result.BytesBuffer = bytes.NewBuffer(nil)
	result.MultiWriter = io.MultiWriter(result.BytesBuffer, realWriter)
	return result
}

// Header TODO
func (w *ResponseSnifferingWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// WriteHeader TODO
func (w *ResponseSnifferingWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write TODO
func (w *ResponseSnifferingWriter) Write(b []byte) (n int, err error) {
	n, err = w.MultiWriter.Write(b)
	return
}
