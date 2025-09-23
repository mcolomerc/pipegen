package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Simulates a minimal subset of Flink SQL Gateway REST API as per nightlies docs:
// - POST /v1/sessions -> create session
// - POST /v1/sessions/{id}/statements -> submit statement returns operationHandle
// - GET  /v1/sessions/{id}/operations/{op}/status -> returns status progression
// - GET  /v1/sessions/{id}/operations/{op}/result/0 (optional for error details)

func TestDeployStatement_Success(t *testing.T) {
	fd := NewFlinkDeployer(&Config{FlinkURL: "http://unused:8081"})
	var sessionCreates int32
	var opPolls int32
	sessionID := "sess-123"
	opHandle := "op-abc"
	// Fake gateway server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
			atomic.AddInt32(&sessionCreates, 1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"sessionHandle": sessionID})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/sessions/") && strings.HasSuffix(r.URL.Path, "/statements"):
			// Accept any statement
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"operationHandle": opHandle})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/sessions/") && strings.Contains(r.URL.Path, "/operations/") && strings.HasSuffix(r.URL.Path, "/status"):
			n := atomic.AddInt32(&opPolls, 1)
			status := "PENDING"
			if n >= 2 { // finish on 2nd poll
				status = "FINISHED"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	// Point FlinkURL at server base, ensuring gateway replacement logic keeps same host (no port change performed)
	fd.config.FlinkURL = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Create session manually via helper under test
	sid, err := fd.createSessionWithRetry(ctx, "pipegen-test", 3, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("session creation failed: %v", err)
	}
	if sid != sessionID {
		t.Fatalf("unexpected session id %s", sid)
	}
	fd.sessionID = sid
	// Deploy one statement
	depID, err := fd.deployStatement(ctx, "create-table", "CREATE TABLE x (id STRING) WITH ('connector'='blackhole')")
	if err != nil {
		t.Fatalf("deployStatement returned error: %v", err)
	}
	if depID != "create-table" { // deployStatement returns name on success
		t.Fatalf("expected deployment id 'create-table', got %s", depID)
	}
	if atomic.LoadInt32(&opPolls) < 2 {
		t.Fatalf("expected at least 2 polls, got %d", opPolls)
	}
}

func TestDeployStatement_ErrorFlow(t *testing.T) {
	fd := NewFlinkDeployer(&Config{FlinkURL: "http://unused:8081"})
	sessionID := "sess-err"
	opHandle := "op-err"
	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
			_ = json.NewEncoder(w).Encode(map[string]string{"sessionHandle": sessionID})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/statements"):
			_ = json.NewEncoder(w).Encode(map[string]string{"operationHandle": opHandle})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
			atomic.AddInt32(&polls, 1)
			// Immediately return ERROR
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": "Validation failed"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/result/0"):
			// Provide detailed error payload
			w.WriteHeader(200)
			_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Column mismatch"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	fd.config.FlinkURL = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sid, err := fd.createSessionWithRetry(ctx, "pipegen-test", 2, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("session creation failed: %v", err)
	}
	fd.sessionID = sid
	if _, err := fd.deployStatement(ctx, "bad-statement", "CREATE TABLE broken"); err == nil {
		t.Fatalf("expected error from deployStatement")
	}
	if atomic.LoadInt32(&polls) == 0 {
		t.Fatalf("expected at least one poll")
	}
}
