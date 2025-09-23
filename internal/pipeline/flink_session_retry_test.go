package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// Test createSessionWithRetry handles transient failures then succeeds
func TestCreateSessionWithRetry_SucceedsAfterRetries(t *testing.T) {
	fd := NewFlinkDeployer(&Config{FlinkURL: "http://unused:8081"})

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/sessions" {
			n := atomic.AddInt32(&attempts, 1)
			if n < 3 { // first two attempts fail
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"sessionHandle":"sess-ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// Point FlinkURL such that port replacement produces the test server URL
	// Replace 8081 with 8083 logic is used; mimic that shape
	fd.config.FlinkURL = srv.URL // includes random port; logic will not replace 8081

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := fd.createSessionWithRetry(ctx, "pipegen-test", 5, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if id != "sess-ok" {
		t.Fatalf("expected session id 'sess-ok', got %s", id)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

// Test createSessionWithRetry exhausts attempts on persistent failure
func TestCreateSessionWithRetry_Fails(t *testing.T) {
	fd := NewFlinkDeployer(&Config{FlinkURL: "http://unused:8081"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	fd.config.FlinkURL = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := fd.createSessionWithRetry(ctx, "pipegen-test", 3, 50*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
