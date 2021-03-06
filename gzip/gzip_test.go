package gzip

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	gzipTestString              = "Foobar Wibble Content"
	gzipTestWebSocketKey        = "Test"
	gzipInvalidCompressionLevel = 11
)

func testHTTPContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, gzipTestString)
}

func Test_ServeHTTP_Compressed(t *testing.T) {
	gzipHandler := Default()
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	body, _ := ioutil.ReadAll(gr)

	if string(body) != gzipTestString {
		t.Fail()
	}
}

func Test_ServeHTTP_NoCompression(t *testing.T) {
	gzipHandler := Default()
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	if w.Body.String() != gzipTestString {
		t.Fail()
	}
}

func Test_ServeHTTP_CompressionWithNoGzipHeader(t *testing.T) {
	gzipHandler := Default()
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	if w.Body.String() != gzipTestString {
		t.Fail()
	}
}

func Test_ServeHTTP_InvalidCompressionLevel(t *testing.T) {
	gzipHandler := New(gzipInvalidCompressionLevel, nil)
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	if w.Body.String() != gzipTestString {
		t.Fail()
	}
}

func Test_ServeHTTP_WebSocketConnection(t *testing.T) {
	gzipHandler := Default()
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(headerAcceptEncoding, encodingGzip)
	req.Header.Set(headerSecWebSocketKey, gzipTestWebSocketKey)

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	if w.Body.String() != gzipTestString {
		t.Fail()
	}
}

func Test_ServeHTTP_AllowCompressionFunc_false(t *testing.T) {
	gzipHandler := New(gzip.DefaultCompression,
		func(w http.ResponseWriter, r *http.Request) bool {
			return false
		},
	)
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	if w.Body.String() != gzipTestString {
		t.Fail()
	}
}

func Test_ServeHTTP_AllowCompressionFunc_true(t *testing.T) {
	gzipHandler := New(gzip.DefaultCompression,
		func(w http.ResponseWriter, r *http.Request) bool {
			return true
		},
	)
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	gzipHandler.ServeHTTP(w, req, testHTTPContent)

	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	body, _ := ioutil.ReadAll(gr)

	if string(body) != gzipTestString {
		t.Fail()
	}
}
