package pipeline

import (
	"context"
	"encoding/json"
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
	globalMode     bool // New field to enable global table creation
}

// NewFlinkDeployer creates a new FlinkSQL deployer
func NewFlinkDeployer(config *Config) *FlinkDeployer {
	fd := &FlinkDeployer{
		config: config,
	}
	fd.stopDeployment = fd.defaultStopDeployment
	return fd
}

// NewFlinkDeployerGlobal creates a new FlinkSQL deployer in global mode
func NewFlinkDeployerGlobal(config *Config) *FlinkDeployer {
	fd := &FlinkDeployer{
		config:     config,
		globalMode: true,
	}
	fd.stopDeployment = fd.defaultStopDeployment
	return fd
}

// Deploy executes FlinkSQL statements in Confluent Cloud
func (fd *FlinkDeployer) Deploy(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	fmt.Printf("⚡ Deploying %d FlinkSQL statements...\n", len(statements))

	if fd.globalMode {
		return fd.deployGlobal(ctx, statements, resources)
	}

	// Original session-based implementation
	return fd.deploySessionBased(ctx, statements, resources)
}

// deployGlobal uses a well-known global session for table creation
func (fd *FlinkDeployer) deployGlobal(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	fmt.Printf("🌍 Using global session approach for table deployment...\n")

	// Use a well-known session name that can be reused across pipeline runs
	globalSessionName := "pipegen-global-session"

	// Try to get or create the global session
	sessionID, err := fd.getOrCreateGlobalSession(ctx, globalSessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create global session: %w", err)
	}

	fd.sessionID = sessionID
	fmt.Printf("📍 Using global session ID: %s\n", sessionID)

	var deploymentIDs []string

	for i, stmt := range statements {
		fmt.Printf("📝 Deploying statement %d globally: %s\n", i+1, stmt.Name)

		// Substitute variables in SQL statement
		processedSQL := fd.substituteVariables(stmt.Content, resources)

		// Deploy the statement using the global session
		deploymentID, err := fd.deployStatement(ctx, stmt.Name, processedSQL)
		if err != nil {
			return deploymentIDs, fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		deploymentIDs = append(deploymentIDs, deploymentID)
		fmt.Printf("  ✅ Deployed globally with ID: %s\n", deploymentID)

		// Small delay between statements to avoid overwhelming the API
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("🎉 All %d statements deployed successfully in global session!\n", len(statements))
	return deploymentIDs, nil
}

// deploySessionBased uses the original session-per-pipeline approach
func (fd *FlinkDeployer) deploySessionBased(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	var deploymentIDs []string

	// Create a session once for all statements
	// Use SQL Gateway URL (typically FlinkURL with port 8083)
	sqlGatewayURL := strings.Replace(fd.config.FlinkURL, "8081", "8083", 1)
	sessionEndpoint := fmt.Sprintf("%s/v1/sessions", sqlGatewayURL)
	sessionReqBody := `{"sessionName": "pipegen-session", "properties": {}}`
	sessionReq, err := http.NewRequestWithContext(ctx, "POST", sessionEndpoint, strings.NewReader(sessionReqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create session request: %w", err)
	}
	sessionReq.Header.Set("Content-Type", "application/json")
	sessionResp, err := http.DefaultClient.Do(sessionReq)
	if err != nil {
		fmt.Printf("⚠️  Session creation request failed: %v\n", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() {
		if err := sessionResp.Body.Close(); err != nil {
			fmt.Printf("failed to close sessionResp.Body: %v\n", err)
		}
	}()
	sessionBody, err := io.ReadAll(sessionResp.Body)
	if err != nil {
		fmt.Printf("⚠️  Failed to read session response: %v\n", err)
		return nil, fmt.Errorf("failed to read session response: %w", err)
	}
	if sessionResp.StatusCode != 200 {
		fmt.Printf("⚠️  Session creation failed with status %d: %s\n", sessionResp.StatusCode, string(sessionBody))
		return nil, fmt.Errorf("flink SQL Gateway session creation failed: %s", string(sessionBody))
	}
	fd.sessionID = extractSessionID(string(sessionBody))
	if fd.sessionID == "" {
		fmt.Printf("⚠️  Could not extract session ID from response: %s\n", string(sessionBody))
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

		// Small delay between statements to avoid overwhelming the API
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("🎉 All %d statements deployed successfully!\n", len(statements))
	return deploymentIDs, nil
}

// getOrCreateGlobalSession gets an existing global session or creates a new one
func (fd *FlinkDeployer) getOrCreateGlobalSession(ctx context.Context, sessionName string) (string, error) {
	sqlGatewayURL := strings.Replace(fd.config.FlinkURL, "8081", "8083", 1)

	// First, try to list existing sessions to see if our global session exists
	listSessionsEndpoint := fmt.Sprintf("%s/v1/sessions", sqlGatewayURL)
	listReq, err := http.NewRequestWithContext(ctx, "GET", listSessionsEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create list sessions request: %w", err)
	}

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}
	defer func() {
		if err := listResp.Body.Close(); err != nil {
			fmt.Printf("failed to close list response body: %v\n", err)
		}
	}()

	if listResp.StatusCode == 200 {
		listBody, err := io.ReadAll(listResp.Body)
		if err == nil {
			fmt.Printf("[Flink SQL Gateway] Current sessions: %s\n", string(listBody))

			// Try to find existing global session
			if existingSessionID := extractExistingSessionID(string(listBody), sessionName); existingSessionID != "" {
				fmt.Printf("🔄 Reusing existing global session: %s\n", existingSessionID)
				return existingSessionID, nil
			}
		}
	}

	// Create new global session
	fmt.Printf("🆕 Creating new global session: %s\n", sessionName)
	sessionEndpoint := fmt.Sprintf("%s/v1/sessions", sqlGatewayURL)
	sessionReqBody := fmt.Sprintf(`{"sessionName": "%s", "properties": {}}`, sessionName)
	sessionReq, err := http.NewRequestWithContext(ctx, "POST", sessionEndpoint, strings.NewReader(sessionReqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create session request: %w", err)
	}
	sessionReq.Header.Set("Content-Type", "application/json")
	sessionResp, err := http.DefaultClient.Do(sessionReq)
	if err != nil {
		fmt.Printf("[Flink SQL Gateway] Global session creation request failed: %v\n", err)
		return "", fmt.Errorf("failed to create global session: %w", err)
	}
	defer func() {
		if err := sessionResp.Body.Close(); err != nil {
			fmt.Printf("failed to close session response body: %v\n", err)
		}
	}()

	sessionBody, err := io.ReadAll(sessionResp.Body)
	if err != nil {
		fmt.Printf("[Flink SQL Gateway] Failed to read global session response: %v\n", err)
		return "", fmt.Errorf("failed to read global session response: %w", err)
	}
	fmt.Printf("[Flink SQL Gateway] Global session creation response: %s\n", string(sessionBody))
	if sessionResp.StatusCode != 200 {
		fmt.Printf("[Flink SQL Gateway] Global session creation failed with status %d: %s\n", sessionResp.StatusCode, string(sessionBody))
		return "", fmt.Errorf("flink SQL Gateway global session creation failed: %s", string(sessionBody))
	}

	sessionID := extractSessionID(string(sessionBody))
	if sessionID == "" {
		fmt.Printf("[Flink SQL Gateway] Could not extract session ID from global session response: %s\n", string(sessionBody))
		return "", fmt.Errorf("could not extract session ID from global session response: %s", string(sessionBody))
	}

	return sessionID, nil
}

// extractExistingSessionID tries to find an existing session with the given name
func extractExistingSessionID(responseBody, sessionName string) string {
	// This would need to parse the JSON response and look for sessions with the matching name
	// For now, return empty string to always create new session
	// TODO: Implement proper JSON parsing to find existing sessions
	return ""
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
	mode := "session-based"
	if fd.globalMode {
		mode = "global"
	}
	fmt.Printf("    🚀 Deploying SQL statement '%s' to Flink SQL Gateway API (%s workflow)...\n", name, mode)

	// 2. Submit statement to session (reuse sessionID)
	if fd.sessionID == "" {
		return "", fmt.Errorf("no sessionID available for statement deployment")
	}
	sqlGatewayURL := strings.Replace(fd.config.FlinkURL, "8081", "8083", 1)
	statementEndpoint := fmt.Sprintf("%s/v1/sessions/%s/statements", sqlGatewayURL, fd.sessionID)
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
		fmt.Printf("⚠️  Statement submission failed with status %d: %s\n", statementResp.StatusCode, string(statementBody))
		return "", fmt.Errorf("flink SQL Gateway returned status %d: %s", statementResp.StatusCode, string(statementBody))
	}

	operationHandle := extractOperationHandle(string(statementBody))
	if operationHandle == "" {
		return "", fmt.Errorf("could not extract operationHandle from response: %s", string(statementBody))
	}

	// 3. Poll operation status until finished
	opStatusEndpoint := fmt.Sprintf("%s/v1/sessions/%s/operations/%s/status", sqlGatewayURL, fd.sessionID, operationHandle)
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
			fmt.Printf("⚠️  Failed to read operation status response for '%s': %v\n", name, err)
			return "", fmt.Errorf("failed to read operation status response: %w", err)
		}
		opStatus = extractOperationStatus(string(body))
		opError = extractOperationError(string(body))
		if opStatus == "FINISHED" {
			fmt.Printf("    ✅ SQL statement '%s' executed successfully.\n", name)
			return name, nil
		}
		if opStatus == "ERROR" || opError != "" {
			// Attempt to retrieve detailed error via helper (handles /result/0 fallback)
			if details, derr := fd.fetchOperationResult(ctx, sqlGatewayURL, operationHandle); derr == nil && details != "" {
				fmt.Printf("⚠️  Error details from result endpoint: %s\n", details)
			} else if derr != nil {
				fmt.Printf("⚠️  Result detail retrieval failed: %v\n", derr)
			}
			fmt.Printf("⚠️  ERROR deploying statement '%s': %s\n", name, opError)
			return "", fmt.Errorf("SQL statement '%s' failed: %s", name, opError)
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

// fetchOperationResult attempts to retrieve the operation result (for error details)
// It first tries the 0-based result endpoint introduced in newer Flink Gateway APIs:
//
//	/v1/sessions/{session}/operations/{op}/result/0
//
// Falls back to legacy endpoint without the trailing /0 if the first returns 404.
// Performs a couple of quick retries in case the result materializes slightly after ERROR status.
func (fd *FlinkDeployer) fetchOperationResult(ctx context.Context, baseURL, operationHandle string) (string, error) {
	if fd.sessionID == "" {
		return "", fmt.Errorf("sessionID not set for operation result fetch")
	}
	primary := fmt.Sprintf("%s/v1/sessions/%s/operations/%s/result/0", baseURL, fd.sessionID, operationHandle)
	legacy := fmt.Sprintf("%s/v1/sessions/%s/operations/%s/result", baseURL, fd.sessionID, operationHandle)

	endpoints := []string{primary, legacy}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ { // 4 attempts total (approx <=2s)
		for idx, ep := range endpoints {
			req, err := http.NewRequestWithContext(ctx, "GET", ep, nil)
			if err != nil {
				lastErr = err
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				lastErr = err
				continue
			}
			body, rerr := io.ReadAll(resp.Body)
			if cerr := resp.Body.Close(); cerr != nil {
				fmt.Printf("failed to close result body: %v\n", cerr)
			}
			if rerr != nil {
				lastErr = rerr
				continue
			}
			if resp.StatusCode == 200 {
				return string(body), nil
			}
			// If first endpoint (primary) returned 404, try legacy within same attempt; others break
			if resp.StatusCode == 404 && idx == 0 {
				continue
			}
			lastErr = fmt.Errorf("result endpoint %s returned %d", ep, resp.StatusCode)
		}
		// brief sleep before next attempt
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no result body available")
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
	fmt.Printf("    🛑 Stopping deployment: %s\n", deploymentID)

	// Create a new context with timeout for cleanup operations to avoid cancellation issues
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all running jobs and cancel them (since deploymentID is just a statement name,
	// we can't match by name easily, so we cancel all running jobs)
	jobsEndpoint := fmt.Sprintf("%s/jobs", fd.config.FlinkURL)
	jobsResp, err := http.Get(jobsEndpoint)
	if err != nil {
		fmt.Printf("    ⚠️  Warning: Failed to get job list for cleanup: %v\n", err)
		return nil // Don't fail completely, just warn
	}
	defer func() { _ = jobsResp.Body.Close() }()

	if jobsResp.StatusCode != http.StatusOK {
		fmt.Printf("    ⚠️  Warning: Failed to get job list, status: %d\n", jobsResp.StatusCode)
		return nil
	}

	jobsBody, err := io.ReadAll(jobsResp.Body)
	if err != nil {
		fmt.Printf("    ⚠️  Warning: Failed to read jobs response: %v\n", err)
		return nil
	}

	var jobsResponse struct {
		Jobs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(jobsBody, &jobsResponse); err != nil {
		fmt.Printf("    ⚠️  Warning: Failed to parse jobs response: %v\n", err)
		return nil
	}

	// Cancel all running jobs
	var cancelledJobs []string
	for _, job := range jobsResponse.Jobs {
		if job.Status == "RUNNING" || job.Status == "CREATED" || job.Status == "RESTARTING" {
			// Cancel the job using the cleanup context
			jobCancelEndpoint := fmt.Sprintf("%s/jobs/%s", fd.config.FlinkURL, job.ID)
			cancelReq, err := http.NewRequestWithContext(cleanupCtx, "PATCH",
				jobCancelEndpoint, nil)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to create cancel request for job %s: %v\n", job.ID, err)
				continue
			}

			cancelResp, err := http.DefaultClient.Do(cancelReq)
			if err != nil {
				fmt.Printf("    ⚠️  Warning: Failed to cancel job %s: %v\n", job.ID, err)
				continue
			}
			_ = cancelResp.Body.Close()

			if cancelResp.StatusCode == http.StatusAccepted {
				cancelledJobs = append(cancelledJobs, job.ID)
				fmt.Printf("    ✅ Cancelled job with ID: %s\n", job.ID)
			} else {
				fmt.Printf("    ⚠️  Warning: Failed to cancel job %s, status: %d\n", job.ID, cancelResp.StatusCode)
			}
		}
	}

	// Wait a moment for jobs to be cancelled
	if len(cancelledJobs) > 0 {
		fmt.Printf("    ⏳ Waiting for %d job(s) to be cancelled...\n", len(cancelledJobs))
		time.Sleep(2 * time.Second)
	}

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

// CancelAllRunningJobs cancels all currently running Flink jobs
func CancelAllRunningJobs(ctx context.Context, flinkURL string) error {
	fmt.Println("🧹 Cancelling all running Flink jobs...")

	// Create a new context with timeout for cleanup operations to avoid cancellation issues
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get all jobs
	jobsEndpoint := fmt.Sprintf("%s/jobs", flinkURL)
	jobsResp, err := http.Get(jobsEndpoint)
	if err != nil {
		return fmt.Errorf("failed to get job list: %w", err)
	}
	defer func() { _ = jobsResp.Body.Close() }()

	if jobsResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get job list, status: %d", jobsResp.StatusCode)
	}

	jobsBody, err := io.ReadAll(jobsResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read jobs response: %w", err)
	}

	var jobsResponse struct {
		Jobs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(jobsBody, &jobsResponse); err != nil {
		return fmt.Errorf("failed to parse jobs response: %w", err)
	}

	var cancelledCount int
	for _, job := range jobsResponse.Jobs {
		if job.Status == "RUNNING" || job.Status == "CREATED" || job.Status == "RESTARTING" {
			// Cancel the job using the cleanup context
			jobCancelEndpoint := fmt.Sprintf("%s/jobs/%s", flinkURL, job.ID)
			cancelReq, err := http.NewRequestWithContext(cleanupCtx, "PATCH",
				jobCancelEndpoint, nil)
			if err != nil {
				fmt.Printf("  ⚠️  Warning: Failed to create cancel request for job %s: %v\n", job.ID, err)
				continue
			}

			cancelResp, err := http.DefaultClient.Do(cancelReq)
			if err != nil {
				fmt.Printf("  ⚠️  Warning: Failed to cancel job %s: %v\n", job.ID, err)
				continue
			}
			_ = cancelResp.Body.Close()

			if cancelResp.StatusCode == http.StatusAccepted {
				cancelledCount++
				fmt.Printf("  ✅ Cancelled job with ID: %s\n", job.ID)
			} else {
				fmt.Printf("  ⚠️  Warning: Failed to cancel job %s, status: %d\n", job.ID, cancelResp.StatusCode)
			}
		}
	}

	if cancelledCount > 0 {
		fmt.Printf("⏳ Waiting for %d job(s) to be cancelled...\n", cancelledCount)
		time.Sleep(3 * time.Second)
		fmt.Printf("✅ Cancelled %d Flink jobs\n", cancelledCount)
	} else {
		fmt.Println("ℹ️  No running jobs found to cancel")
	}

	return nil
}
