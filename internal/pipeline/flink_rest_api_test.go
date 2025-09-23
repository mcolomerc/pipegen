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
	fd.config.SQLGatewayURL = ""
	fd.config.Normalize()
	fd.config.SQLGatewayURL = "" // force re-derive
	fd.config.Normalize()
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
	fd.config.SQLGatewayURL = ""
	fd.config.Normalize()
	fd.config.SQLGatewayURL = ""
	fd.config.Normalize()
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

// TestFetchOperationResult_Success simulates FINISHED status where a subsequent call to fetchOperationResult
// should obtain result payload (e.g., catalog info or query result). We exercise both primary (/result/0)
// and legacy (/result) fallback by first letting primary succeed, then legacy path in a second subtest.
func TestFetchOperationResult_Success(t *testing.T) {
	t.Run("primary-endpoint", func(t *testing.T) {
		fd := NewFlinkDeployer(&Config{FlinkURL: "http://unused:8081"})
		sessionID := "sess-ok"
		opHandle := "op-ok"
		// Gateway server: create session, submit statement, FINISHED status, primary result endpoint returns payload
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
				_ = json.NewEncoder(w).Encode(map[string]string{"sessionHandle": sessionID})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/statements"):
				_ = json.NewEncoder(w).Encode(map[string]string{"operationHandle": opHandle})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "FINISHED"})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/result/0"):
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]string{{"col": "val"}}})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()
		fd.config.FlinkURL = srv.URL
		fd.config.SQLGatewayURL = ""
		fd.config.Normalize()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		sid, err := fd.createSessionWithRetry(ctx, "pipegen-test", 2, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("session creation failed: %v", err)
		}
		fd.sessionID = sid
		// Deploy (FINISHED immediately)
		if _, err := fd.deployStatement(ctx, "finished-statement", "CREATE TABLE done (id STRING)"); err != nil {
			t.Fatalf("deployStatement failed: %v", err)
		}
		// Explicitly call fetchOperationResult to validate direct retrieval
		res, err := fd.fetchOperationResult(ctx, fd.config.SQLGatewayURL, opHandle)
		if err != nil {
			t.Fatalf("fetchOperationResult error: %v", err)
		}
		if !strings.Contains(res, "\"col\":\"val\"") {
			t.Fatalf("expected result payload, got: %s", res)
		}
	})

	t.Run("legacy-endpoint-fallback", func(t *testing.T) {
		fd := NewFlinkDeployer(&Config{FlinkURL: "http://unused:8081"})
		sessionID := "sess-legacy"
		opHandle := "op-legacy"
		// Server returns 404 for primary /result/0 then serves payload at legacy /result
		var primaryHits int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
				_ = json.NewEncoder(w).Encode(map[string]string{"sessionHandle": sessionID})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/statements"):
				_ = json.NewEncoder(w).Encode(map[string]string{"operationHandle": opHandle})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "FINISHED"})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/result/0"):
				atomic.AddInt32(&primaryHits, 1)
				http.Error(w, "not found", http.StatusNotFound)
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/result"):
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]int{{"x": 1}}})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()
		fd.config.FlinkURL = srv.URL
		fd.config.SQLGatewayURL = ""
		fd.config.Normalize()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		sid, err := fd.createSessionWithRetry(ctx, "pipegen-test", 2, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("session creation failed: %v", err)
		}
		fd.sessionID = sid
		if _, err := fd.deployStatement(ctx, "finished-statement", "CREATE TABLE done (id STRING)"); err != nil {
			t.Fatalf("deployStatement failed: %v", err)
		}
		res, err := fd.fetchOperationResult(ctx, fd.config.SQLGatewayURL, opHandle)
		if err != nil {
			t.Fatalf("fetchOperationResult error: %v", err)
		}
		if !strings.Contains(res, "\"x\":1") {
			t.Fatalf("expected legacy result payload, got: %s", res)
		}
		if atomic.LoadInt32(&primaryHits) == 0 {
			t.Fatalf("expected at least one primary /result/0 attempt")
		}
	})
}
