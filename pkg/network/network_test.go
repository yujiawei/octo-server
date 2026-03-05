package network

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostForWWWFormForBytres_ClosesResponseBody(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Call the function
	body, err := PostForWWWFormForBytres(server.URL, map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("expected body %q, got %q", `{"status":"ok"}`, string(body))
	}
}

func TestPostForWWWFormForAll_ClosesResponseBody(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Call the function
	body, err := PostForWWWFormForAll(server.URL, strings.NewReader("key=value"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("expected body %q, got %q", `{"status":"ok"}`, string(body))
	}
}

func TestPostForWWWFormForBytres_NonOKStatus(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer server.Close()

	// Call the function
	body, err := PostForWWWFormForBytres(server.URL, map[string]string{"key": "value"}, nil)
	if err == nil {
		t.Fatal("expected error for non-OK status")
	}

	// Body should still be returned
	if string(body) != `error` {
		t.Errorf("expected body %q, got %q", `error`, string(body))
	}
}

func TestPostForWWWFormForAll_NonOKStatus(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer server.Close()

	// Call the function - note this returns nil body on error (different from ForBytres)
	body, err := PostForWWWFormForAll(server.URL, strings.NewReader("key=value"), nil)
	if err == nil {
		t.Fatal("expected error for non-OK status")
	}

	// Body should be nil for this function
	if body != nil {
		t.Errorf("expected nil body, got %q", string(body))
	}
}

// verifyResponseBodyClosed is a helper to track if response body is closed
type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}
