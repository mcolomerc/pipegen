// filepath: internal/pipeline/flink_test.go
package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test NewFlinkDeployer returns a non-nil deployer with correct config
func TestNewFlinkDeployer(t *testing.T) {
	cfg := &Config{BootstrapServers: "bs", SchemaRegistryURL: "sr"}
	deployer := NewFlinkDeployer(cfg)
	assert.NotNil(t, deployer)
	assert.Equal(t, cfg, deployer.config)
}

// Test substituteVariables replaces all placeholders correctly
func TestFlinkDeployer_SubstituteVariables(t *testing.T) {
	cfg := &Config{BootstrapServers: "bs", SchemaRegistryURL: "sr"}
	res := &Resources{InputTopic: "in", OutputTopic: "out"}
	deployer := NewFlinkDeployer(cfg)
	sql := "CREATE TABLE ${INPUT_TOPIC} WITH ('bootstrap.servers' = '${BOOTSTRAP_SERVERS}', 'schema.registry.url' = '${SCHEMA_REGISTRY_URL}', 'output' = '${OUTPUT_TOPIC}');"
	expected := "CREATE TABLE in WITH ('bootstrap.servers' = 'bs', 'schema.registry.url' = 'sr', 'output' = 'out');"
	result := deployer.substituteVariables(sql, res)
	assert.Equal(t, expected, result)
}

// Tests for new JSON parsing helpers
func TestJSONParsingHelpers(t *testing.T) {
	t.Run("parseSessionID valid", func(t *testing.T) {
		id, err := parseSessionID([]byte(`{"sessionHandle":"abc123"}`))
		assert.NoError(t, err)
		assert.Equal(t, "abc123", id)
	})
	t.Run("parseSessionID missing", func(t *testing.T) {
		_, err := parseSessionID([]byte(`{}`))
		assert.Error(t, err)
	})
	t.Run("parseStatementOperationHandle valid", func(t *testing.T) {
		op, err := parseStatementOperationHandle([]byte(`{"operationHandle":"op456"}`))
		assert.NoError(t, err)
		assert.Equal(t, "op456", op)
	})
	t.Run("parseStatementOperationHandle missing", func(t *testing.T) {
		_, err := parseStatementOperationHandle([]byte(`{}`))
		assert.Error(t, err)
	})
	t.Run("parseOperationStatus finished", func(t *testing.T) {
		st, err := parseOperationStatus([]byte(`{"status":"FINISHED"}`))
		assert.NoError(t, err)
		assert.Equal(t, "FINISHED", st.Status)
		assert.Equal(t, "", st.Error)
	})
	t.Run("parseOperationStatus error with message", func(t *testing.T) {
		st, err := parseOperationStatus([]byte(`{"status":"ERROR","error":"boom"}`))
		assert.NoError(t, err)
		assert.Equal(t, "ERROR", st.Status)
		assert.Equal(t, "boom", st.Error)
	})
	t.Run("parseOperationStatus missing status", func(t *testing.T) {
		_, err := parseOperationStatus([]byte(`{"error":"boom"}`))
		assert.Error(t, err)
	})
}

// Test escapeJSONString escapes quotes, backslashes, and newlines
func TestEscapeJSONString(t *testing.T) {
	input := `foo"bar\baz
qux`
	expected := `foo\"bar\\baz\nqux`
	got := escapeJSONString(input)
	assert.Equal(t, expected, got)
}

// Test truncateSQL truncates long SQL and leaves short SQL unchanged
func TestFlinkDeployer_TruncateSQL(t *testing.T) {
	deployer := NewFlinkDeployer(&Config{})
	short := "SELECT * FROM foo"
	long := "SELECT " + string(make([]byte, 200)) + " FROM bar"
	assert.Equal(t, short, deployer.truncateSQL(short))
	trunc := deployer.truncateSQL(long)
	assert.True(t, len(trunc) < len(long))
	assert.Contains(t, trunc, "...")
}

// Test Cleanup calls stopDeployment for each deploymentID
func TestFlinkDeployer_Cleanup(t *testing.T) {
	deployer := NewFlinkDeployer(&Config{})
	called := make(map[string]bool)
	// Patch stopDeployment for test
	originalStopDeployment := deployer.stopDeployment
	defer func() { deployer.stopDeployment = originalStopDeployment }() // Restore after test

	deployer.stopDeployment = func(ctx context.Context, deploymentID string) error {
		called[deploymentID] = true
		return nil
	}
	ids := []string{"id1", "id2"}
	err := deployer.Cleanup(context.Background(), ids)
	assert.NoError(t, err)
	assert.True(t, called["id1"])
	assert.True(t, called["id2"])
}

// Test GetDeploymentStatus returns expected stub status
func TestFlinkDeployer_GetDeploymentStatus(t *testing.T) {
	deployer := NewFlinkDeployer(&Config{})
	status, err := deployer.GetDeploymentStatus(context.Background(), "foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", status.ID)
	assert.Equal(t, "RUNNING", status.Status)
	assert.Equal(t, "RUNNING", status.Phase)
}

// Test FlinkAPIClient stub methods
func TestFlinkAPIClient_Stubs(t *testing.T) {
	client := NewFlinkAPIClient("key", "secret")
	id, err := client.ExecuteStatement(context.Background(), "env", "pool", "SELECT 1")
	assert.NoError(t, err)
	assert.Equal(t, "statement-id-123", id)

	err = client.StopStatement(context.Background(), "statement-id-123")
	assert.NoError(t, err)

	status, err := client.GetStatementStatus(context.Background(), "statement-id-123")
	assert.NoError(t, err)
	assert.Equal(t, "statement-id-123", status.ID)
	assert.Equal(t, "RUNNING", status.Status)
	assert.Equal(t, "RUNNING", status.Phase)
}

// Test StatusCallback type signature (compiles)
func TestStatusCallbackType(t *testing.T) {
	var cb StatusCallback = func(statementName, status, phase, deploymentID, errorMsg string) {}
	cb("name", "RUNNING", "DEPLOYING", "id", "")
}

// Test DeploymentStatus struct JSON tags
func TestDeploymentStatusTags(t *testing.T) {
	status := DeploymentStatus{ID: "id", Status: "RUNNING", Phase: "RUNNING", Error: "err"}
	assert.Equal(t, "id", status.ID)
	assert.Equal(t, "RUNNING", status.Status)
	assert.Equal(t, "RUNNING", status.Phase)
	assert.Equal(t, "err", status.Error)
}

// Test fetchOperationResult helper (primary endpoint, fallback, transient success, and failure)
func TestFlinkDeployer_FetchOperationResult(t *testing.T) {
	deployer := NewFlinkDeployer(&Config{})
	deployer.sessionID = "sess1"
	operationHandle := "op123"

	t.Run("primary endpoint success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/sessions/sess1/operations/op123/result/0" {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"results":"primary"}`))
				return
			}
			w.WriteHeader(404)
		}))
		defer srv.Close()
		body, err := deployer.fetchOperationResult(context.Background(), srv.URL, operationHandle)
		assert.NoError(t, err)
		assert.Contains(t, body, "primary")
	})

	t.Run("fallback endpoint success after 404 primary", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/sessions/sess1/operations/op123/result/0":
				w.WriteHeader(404)
			case "/v1/sessions/sess1/operations/op123/result":
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"results":"legacy"}`))
			default:
				w.WriteHeader(404)
			}
		}))
		defer srv.Close()
		body, err := deployer.fetchOperationResult(context.Background(), srv.URL, operationHandle)
		assert.NoError(t, err)
		assert.Contains(t, body, "legacy")
	})

	t.Run("transient primary 404 then success", func(t *testing.T) {
		var hits int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/sessions/sess1/operations/op123/result/0" {
				count := atomic.AddInt32(&hits, 1)
				if count == 1 { // first attempt 404
					w.WriteHeader(404)
					return
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"results":"eventual"}`))
				return
			}
			w.WriteHeader(404)
		}))
		defer srv.Close()
		body, err := deployer.fetchOperationResult(context.Background(), srv.URL, operationHandle)
		assert.NoError(t, err)
		assert.Contains(t, body, "eventual")
		// ensure at least 2 hits on primary
		assert.GreaterOrEqual(t, atomic.LoadInt32(&hits), int32(2))
	})

	t.Run("all endpoints fail", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer srv.Close()
		_, err := deployer.fetchOperationResult(context.Background(), srv.URL, operationHandle)
		assert.Error(t, err)
	})
}
