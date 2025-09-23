package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// minimal stack deployer for test (reuse constructor)

func TestCreateFlinkSession_Retry(t *testing.T) {
	d := NewStackDeployer("/tmp")
	var readinessHits int32
	var postHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sessions" {
			if r.Method == http.MethodGet { // readiness
				n := atomic.AddInt32(&readinessHits, 1)
				if n < 2 { // first readiness fails
					http.Error(w, "warming", http.StatusBadGateway)
					return
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`[]`))
				return
			}
			if r.Method == http.MethodPost { // session create
				n := atomic.AddInt32(&postHits, 1)
				if n < 2 { // first post fails
					http.Error(w, "starting", http.StatusBadGateway)
					return
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"sessionHandle":"local-sess"}`))
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	d.sqlGatewayAddr = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	id, err := d.createFlinkSession(ctx, client)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if id != "local-sess" {
		t.Fatalf("unexpected session id: %s", id)
	}
	if atomic.LoadInt32(&postHits) < 2 {
		t.Fatalf("expected at least 2 POST attempts, got %d", postHits)
	}
	if atomic.LoadInt32(&readinessHits) < 2 {
		t.Fatalf("expected at least 2 readiness attempts, got %d", readinessHits)
	}
}
func TestCreateFlinkSession_Fail(t *testing.T) {
	d := NewStackDeployer("/tmp")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	d.sqlGatewayAddr = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 1 * time.Second}
	_, err := d.createFlinkSession(ctx, client)
	if err == nil {
		t.Fatalf("expected failure")
	}
}

func TestWaitForLocalSQLGatewayReady(t *testing.T) {
	d := NewStackDeployer("/tmp")
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sessions" {
			n := atomic.AddInt32(&hits, 1)
			if n < 2 {
				http.Error(w, "warming", http.StatusBadGateway)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	d.sqlGatewayAddr = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := d.waitForLocalSQLGatewayReady(ctx, 3*time.Second); err != nil {
		t.Fatalf("expected readiness success, got %v", err)
	}
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatalf("expected at least 2 readiness attempts, got %d", hits)
	}
}
