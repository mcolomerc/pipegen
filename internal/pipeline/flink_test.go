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

// Table-driven tests for extractSessionID, extractOperationHandle, extractOperationStatus, extractOperationError
func TestExtractHelpers(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) string
		input    string
		expected string
	}{
		{
			name:     "extractSessionID valid",
			fn:       extractSessionID,
			input:    `{"sessionHandle":"abc123"}`,
			expected: "abc123",
		},
		{
			name:     "extractSessionID missing",
			fn:       extractSessionID,
			input:    `{}`,
			expected: "",
		},
		{
			name:     "extractOperationHandle valid",
			fn:       extractOperationHandle,
			input:    `{"operationHandle":"op456"}`,
			expected: "op456",
		},
		{
			name:     "extractOperationStatus valid",
			fn:       extractOperationStatus,
			input:    `{"status":"FINISHED"}`,
			expected: "FINISHED",
		},
		{
			name:     "extractOperationError valid",
			fn:       extractOperationError,
			input:    `{"error":"failure"}`,
			expected: "failure",
		},
		{
			name:     "extractOperationError missing",
			fn:       extractOperationError,
			input:    `{}`,
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
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
