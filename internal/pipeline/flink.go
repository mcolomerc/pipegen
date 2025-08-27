package pipeline

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"pipegen/internal/types"
	"strings"
	"time"
)

// DeploymentStatus represents the status of a FlinkSQL deployment
type DeploymentStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Phase  string `json:"phase"`
	Error  string `json:"error,omitempty"`
}

// FlinkAPIClient is a stub for the FlinkSQL API client
type FlinkAPIClient struct {
	apiKey    string
	apiSecret string
}

// FlinkDeployer handles FlinkSQL statement deployment via Confluent Cloud API
type FlinkDeployer struct {
	config         *Config
	sessionID      string
	stopDeployment func(ctx context.Context, deploymentID string) error
}

// NewFlinkDeployer creates a new FlinkSQL deployer
func NewFlinkDeployer(config *Config) *FlinkDeployer {
	fd := &FlinkDeployer{
		config: config,
	}
	fd.stopDeployment = fd.defaultStopDeployment
	return fd
}

// Deploy executes FlinkSQL statements in Confluent Cloud
func (fd *FlinkDeployer) Deploy(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	fmt.Printf("⚡ Deploying %d FlinkSQL statements...\n", len(statements))

	var deploymentIDs []string

	// Create a session once for all statements
	sessionEndpoint := "http://localhost:8083/v1/sessions"
	sessionReqBody := `{"sessionName": "pipegen-session", "properties": {}}`
	sessionReq, err := http.NewRequestWithContext(ctx, "POST", sessionEndpoint, strings.NewReader(sessionReqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create session request: %w", err)
	}
	sessionReq.Header.Set("Content-Type", "application/json")
	sessionResp, err := http.DefaultClient.Do(sessionReq)
	if err != nil {
		fmt.Printf("[Flink SQL Gateway] Session creation request failed: %v\n", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() {
		if err := sessionResp.Body.Close(); err != nil {
			fmt.Printf("failed to close sessionResp.Body: %v\n", err)
		}
	}()
	sessionBody, err := io.ReadAll(sessionResp.Body)
	if err != nil {
		fmt.Printf("[Flink SQL Gateway] Failed to read session response: %v\n", err)
		return nil, fmt.Errorf("failed to read session response: %w", err)
	}
	fmt.Printf("[Flink SQL Gateway] Session creation response: %s\n", string(sessionBody))
	if sessionResp.StatusCode != 200 {
		fmt.Printf("[Flink SQL Gateway] Session creation failed with status %d: %s\n", sessionResp.StatusCode, string(sessionBody))
		return nil, fmt.Errorf("flink SQL Gateway session creation failed: %s", string(sessionBody))
	}
	fd.sessionID = extractSessionID(string(sessionBody))
	if fd.sessionID == "" {
		fmt.Printf("[Flink SQL Gateway] Could not extract session ID from response: %s\n", string(sessionBody))
		return nil, fmt.Errorf("could not extract session ID from response: %s", string(sessionBody))
	}

	for i, stmt := range statements {
		fmt.Printf("📝 Deploying statement %d: %s\n", i+1, stmt.Name)

		// Substitute variables in SQL statement
		processedSQL := fd.substituteVariables(stmt.Content, resources)

		// Deploy the statement using the same session
		deploymentID, err := fd.deployStatement(ctx, stmt.Name, processedSQL)
		if err != nil {
			return deploymentIDs, fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		deploymentIDs = append(deploymentIDs, deploymentID)
		fmt.Printf("  ✅ Deployed with ID: %s\n", deploymentID)
	}

	fmt.Printf("✅ All %d statements deployed successfully\n", len(statements))
	return deploymentIDs, nil
}

// StatusCallback is a function type for statement status updates
type StatusCallback func(statementName, status, phase, deploymentID, errorMsg string)

// DeployWithStatusTracking executes FlinkSQL statements with status tracking
func (fd *FlinkDeployer) DeployWithStatusTracking(ctx context.Context, statements []*types.SQLStatement, resources *Resources, statusCallback StatusCallback) ([]string, error) {
	fmt.Printf("⚡ Deploying %d FlinkSQL statements with status tracking...\n", len(statements))

	var deploymentIDs []string

	for i, stmt := range statements {
		fmt.Printf("📝 Deploying statement %d: %s\n", i+1, stmt.Name)

		// Update status to deploying
		if statusCallback != nil {
			statusCallback(stmt.Name, "RUNNING", "DEPLOYING", "", "")
		}

		// Substitute variables in SQL statement
		processedSQL := fd.substituteVariables(stmt.Content, resources)

		// Deploy the statement
		deploymentID, err := fd.deployStatement(ctx, stmt.Name, processedSQL)
		if err != nil {
			// Update status to failed
			if statusCallback != nil {
				statusCallback(stmt.Name, "FAILED", "ERROR", "", err.Error())
			}
			return deploymentIDs, fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		deploymentIDs = append(deploymentIDs, deploymentID)
		fmt.Printf("  ✅ Deployed with ID: %s\n", deploymentID)

		// Update status to running
		if statusCallback != nil {
			statusCallback(stmt.Name, "RUNNING", "ACTIVE", deploymentID, "")
		}
	}

	fmt.Printf("✅ All %d statements deployed successfully\n", len(statements))
	return deploymentIDs, nil
}

// deployStatement deploys a single FlinkSQL statement
func (fd *FlinkDeployer) deployStatement(ctx context.Context, name, sql string) (string, error) {
	fmt.Printf("    🚀 Deploying SQL statement '%s' to Flink SQL Gateway API (session workflow)...\n", name)

	// 2. Submit statement to session (reuse sessionID)
	if fd.sessionID == "" {
		return "", fmt.Errorf("no sessionID available for statement deployment")
	}
	statementEndpoint := fmt.Sprintf("http://localhost:8083/v1/sessions/%s/statements", fd.sessionID)
	statementReqBody := fmt.Sprintf(`{"statement": "%s"}`, escapeJSONString(sql))
	statementReq, err := http.NewRequestWithContext(ctx, "POST", statementEndpoint, strings.NewReader(statementReqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create statement request: %w", err)
	}
	statementReq.Header.Set("Content-Type", "application/json")
	statementResp, err := http.DefaultClient.Do(statementReq)
	if err != nil {
		return "", fmt.Errorf("failed to send SQL statement to Flink SQL Gateway: %w", err)
	}
	defer func() {
		if err := statementResp.Body.Close(); err != nil {
			fmt.Printf("failed to close statementResp.Body: %v\n", err)
		}
	}()
	statementBody, err := io.ReadAll(statementResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read statement response: %w", err)
	}
	if statementResp.StatusCode != 200 {
		fmt.Printf("[Flink SQL Gateway] Statement submission failed with status %d: %s\n", statementResp.StatusCode, string(statementBody))
		return "", fmt.Errorf("flink SQL Gateway returned status %d: %s", statementResp.StatusCode, string(statementBody))
	}

	// Log the full statement submission response for debugging
	fmt.Printf("[Flink SQL Gateway] Statement submission response for '%s': %s\n", name, string(statementBody))
	operationHandle := extractOperationHandle(string(statementBody))
	if operationHandle == "" {
		return "", fmt.Errorf("could not extract operationHandle from response: %s", string(statementBody))
	}

	// 3. Poll operation status until finished
	opStatusEndpoint := fmt.Sprintf("http://localhost:8083/v1/sessions/%s/operations/%s/status", fd.sessionID, operationHandle)
	var opStatus string
	var opError string
	for i := 0; i < 30; i++ { // up to 30 attempts, 1s apart
		select {
		case <-ctx.Done():
			fmt.Printf("[Flink SQL Gateway] Context cancelled while waiting for operation status for '%s'\n", name)
			return "", fmt.Errorf("context cancelled while waiting for operation status")
		default:
		}
		req, err := http.NewRequestWithContext(ctx, "GET", opStatusEndpoint, nil)
		if err != nil {
			fmt.Printf("[Flink SQL Gateway] Failed to create operation status request for '%s': %v\n", name, err)
			return "", fmt.Errorf("failed to create operation status request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("[Flink SQL Gateway] Failed to poll operation status for '%s': %v\n", name, err)
			return "", fmt.Errorf("failed to poll operation status: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close resp.Body: %v\n", err)
		}
		if err != nil {
			fmt.Printf("[Flink SQL Gateway] Failed to read operation status response for '%s': %v\n", name, err)
			return "", fmt.Errorf("failed to read operation status response: %w", err)
		}
		// Log the full response for visibility
		fmt.Printf("[Flink SQL Gateway] Operation status response for '%s': %s\n", name, string(body))
		opStatus = extractOperationStatus(string(body))
		opError = extractOperationError(string(body))
		if opStatus == "FINISHED" {
			fmt.Printf("    ✅ SQL statement '%s' executed successfully.\n", name)
			return name, nil
		}
		if opStatus == "ERROR" || opError != "" {
			fmt.Printf("[Flink SQL Gateway] ERROR deploying statement '%s': %s\nFull response: %s\n", name, opError, string(body))
			return "", fmt.Errorf("SQL statement '%s' failed: %s\nFull response: %s", name, opError, string(body))
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Printf("[Flink SQL Gateway] Statement '%s' did not finish after polling. Last status: %s, error: %s\n", name, opStatus, opError)
	return "", fmt.Errorf("SQL statement '%s' did not finish after polling. Last status: %s, error: %s", name, opStatus, opError)
}

// extractOperationHandle parses the operationHandle from the statement response
func extractOperationHandle(resp string) string {
	idx := strings.Index(resp, "\"operationHandle\":\"")
	if idx == -1 {
		return ""
	}
	start := idx + len("\"operationHandle\":\"")
	end := strings.Index(resp[start:], "\"")
	if end == -1 {
		return ""
	}
	return resp[start : start+end]
}

// extractOperationStatus parses the status from the operation status response
func extractOperationStatus(resp string) string {
	idx := strings.Index(resp, "\"status\":\"")
	if idx == -1 {
		return ""
	}
	start := idx + len("\"status\":\"")
	end := strings.Index(resp[start:], "\"")
	if end == -1 {
		return ""
	}
	return resp[start : start+end]
}

// extractOperationError parses the error from the operation status response
func extractOperationError(resp string) string {
	idx := strings.Index(resp, "\"error\":\"")
	if idx == -1 {
		return ""
	}
	start := idx + len("\"error\":\"")
	end := strings.Index(resp[start:], "\"")
	if end == -1 {
		return ""
	}
	return resp[start : start+end]
}

// extractSessionID parses the session id from the session creation response
func extractSessionID(resp string) string {
	// Updated extraction: look for "sessionHandle":"..."
	idx := strings.Index(resp, "\"sessionHandle\":\"")
	if idx == -1 {
		return ""
	}
	start := idx + len("\"sessionHandle\":\"")
	end := strings.Index(resp[start:], "\"")
	if end == -1 {
		return ""
	}
	return resp[start : start+end]
}

// escapeJSONString escapes quotes and newlines for JSON string
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// writeTempSQLFile writes the SQL content to a temporary file

// runCommand runs a shell command and returns its output

// substituteVariables replaces placeholders in SQL statements with actual values
func (fd *FlinkDeployer) substituteVariables(sql string, resources *Resources) string {
	replacements := map[string]string{
		"${INPUT_TOPIC}":         resources.InputTopic,
		"${OUTPUT_TOPIC}":        resources.OutputTopic,
		"${BOOTSTRAP_SERVERS}":   fd.config.BootstrapServers,
		"${SCHEMA_REGISTRY_URL}": fd.config.SchemaRegistryURL,
	}

	processedSQL := sql
	for placeholder, value := range replacements {
		processedSQL = strings.ReplaceAll(processedSQL, placeholder, value)
	}

	return processedSQL
}

// truncateSQL truncates SQL for display purposes
func (fd *FlinkDeployer) truncateSQL(sql string) string {
	const maxLength = 100
	cleaned := strings.ReplaceAll(strings.TrimSpace(sql), "\n", " ")
	if len(cleaned) > maxLength {
		return cleaned[:maxLength] + "..."
	}
	return cleaned
}

// Cleanup stops and removes FlinkSQL deployments
func (fd *FlinkDeployer) Cleanup(ctx context.Context, deploymentIDs []string) error {
	fmt.Printf("🧹 Cleaning up %d FlinkSQL deployments...\n", len(deploymentIDs))

	for _, deploymentID := range deploymentIDs {
		if err := fd.stopDeployment(ctx, deploymentID); err != nil {
			return fmt.Errorf("failed to stop deployment %s: %w", deploymentID, err)
		}
		fmt.Printf("  ✅ Stopped deployment: %s\n", deploymentID)
	}

	fmt.Printf("✅ All deployments cleaned up\n")
	return nil
}

// defaultStopDeployment is the actual implementation
func (fd *FlinkDeployer) defaultStopDeployment(ctx context.Context, deploymentID string) error {
	// TODO: Implement actual Confluent Cloud FlinkSQL API call
	// This is a placeholder for the actual implementation
	fmt.Printf("    🛑 Stopping deployment: %s\n", deploymentID)
	// Here you would use the Confluent Cloud FlinkSQL API to:
	// 1. Stop the statement execution
	// 2. Clean up associated resources
	// 3. Confirm successful cleanup
	return nil
}

// GetDeploymentStatus checks the status of a FlinkSQL deployment
func (fd *FlinkDeployer) GetDeploymentStatus(ctx context.Context, deploymentID string) (*DeploymentStatus, error) {
	// TODO: Implement actual status check
	return &DeploymentStatus{
		ID:     deploymentID,
		Status: "RUNNING",
		Phase:  "RUNNING",
	}, nil
}

// NewFlinkAPIClient creates a new FlinkSQL API client
func NewFlinkAPIClient(apiKey, apiSecret string) *FlinkAPIClient {
	return &FlinkAPIClient{
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// ExecuteStatement executes a FlinkSQL statement
func (client *FlinkAPIClient) ExecuteStatement(ctx context.Context, environmentID, computePoolID, sql string) (string, error) {
	// TODO: Implement actual HTTP API call to Confluent Cloud
	// This would include:
	// 1. Authentication with API key/secret
	// 2. Proper request formatting
	// 3. Error handling and retries
	// 4. Response parsing

	return "statement-id-123", nil
}

// StopStatement stops a running FlinkSQL statement
func (client *FlinkAPIClient) StopStatement(ctx context.Context, statementID string) error {
	// TODO: Implement actual HTTP API call to stop statement
	return nil
}

// GetStatementStatus gets the status of a FlinkSQL statement
func (client *FlinkAPIClient) GetStatementStatus(ctx context.Context, statementID string) (*DeploymentStatus, error) {
	// TODO: Implement actual HTTP API call to get statement status
	return &DeploymentStatus{
		ID:     statementID,
		Status: "RUNNING",
		Phase:  "RUNNING",
	}, nil
}
