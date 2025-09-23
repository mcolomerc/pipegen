package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	logpkg "pipegen/internal/log"
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
	logger         logpkg.Logger
}

// --- Typed response structs for Flink SQL Gateway ---
type sessionCreateResponse struct {
	SessionHandle string `json:"sessionHandle"`
}

type statementSubmitResponse struct {
	OperationHandle string `json:"operationHandle"`
}

type operationStatusResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// parseSessionID decodes the session creation response and returns the session handle
func parseSessionID(body []byte) (string, error) {
	var resp sessionCreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("unable to decode session response: %w", err)
	}
	if resp.SessionHandle == "" {
		return "", fmt.Errorf("sessionHandle missing in response")
	}
	return resp.SessionHandle, nil
}

// parseStatementOperationHandle decodes the statement submission response
func parseStatementOperationHandle(body []byte) (string, error) {
	var resp statementSubmitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("unable to decode statement response: %w", err)
	}
	if resp.OperationHandle == "" {
		return "", fmt.Errorf("operationHandle missing in response")
	}
	return resp.OperationHandle, nil
}

// parseOperationStatus decodes the operation status response
func parseOperationStatus(body []byte) (*operationStatusResponse, error) {
	var resp operationStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unable to decode operation status: %w", err)
	}
	if resp.Status == "" { // status should always be present
		return nil, fmt.Errorf("status missing in operation status response")
	}
	return &resp, nil
}

// getSQLGatewayURL resolves the SQL Gateway base URL. Preference order:
// 1. Explicit Config.SQLGatewayURL if provided.
// 2. Derive from FlinkURL by replacing :8081 with :8083 (common local convention).
// 3. If no port substitution possible, append :8083 if missing.
func (fd *FlinkDeployer) getSQLGatewayURL() string { return fd.config.SQLGatewayURL }

// waitForSQLGatewayReady performs a lightweight readiness check by polling the sessions list endpoint.
// It tolerates transient connection resets. Returns nil when it receives any HTTP 200 response.
func (fd *FlinkDeployer) waitForSQLGatewayReady(ctx context.Context, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	endpoint := fmt.Sprintf("%s/v1/sessions", baseURL)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for SQL Gateway readiness: %w", ctx.Err())
		default:
		}
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to build readiness request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			fd.logger.Warn("sql gateway readiness transient error", "error", err)
			time.Sleep(750 * time.Millisecond)
			continue
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == 200 { // Ready
			fd.logger.Info("sql gateway readiness confirmed")
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
		fd.logger.Debug("sql gateway readiness non-200", "status", resp.StatusCode)
		time.Sleep(750 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unknown readiness failure")
	}
	return lastErr
}

// NewFlinkDeployer creates a new FlinkSQL deployer
func NewFlinkDeployer(config *Config) *FlinkDeployer {
	fd := &FlinkDeployer{
		config: config,
		logger: logpkg.Global(),
	}
	fd.stopDeployment = fd.defaultStopDeployment
	return fd
}

// NewFlinkDeployerGlobal creates a new FlinkSQL deployer in global mode
func NewFlinkDeployerGlobal(config *Config) *FlinkDeployer {
	fd := &FlinkDeployer{
		config:     config,
		globalMode: true,
		logger:     logpkg.Global(),
	}
	fd.stopDeployment = fd.defaultStopDeployment
	return fd
}

// Deploy executes FlinkSQL statements in Confluent Cloud
func (fd *FlinkDeployer) Deploy(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	fd.logger.Info("deploying flink sql statements", "count", len(statements), "mode", map[bool]string{true: "global", false: "session"}[fd.globalMode])

	if fd.globalMode {
		return fd.deployGlobal(ctx, statements, resources)
	}

	// Original session-based implementation
	return fd.deploySessionBased(ctx, statements, resources)
}

// deployGlobal uses a well-known global session for table creation
func (fd *FlinkDeployer) deployGlobal(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	fd.logger.Info("using global session approach")

	// Use a well-known session name that can be reused across pipeline runs
	globalSessionName := "pipegen-global-session"

	// Try to get or create the global session
	sessionID, err := fd.getOrCreateGlobalSession(ctx, globalSessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create global session: %w", err)
	}

	fd.sessionID = sessionID
	fd.logger.Info("using global session id", "session_id", sessionID)

	var deploymentIDs []string

	for i, stmt := range statements {
		fd.logger.Info("deploying statement globally", "index", i+1, "total", len(statements), "name", stmt.Name)

		// Substitute variables in SQL statement
		processedSQL := fd.substituteVariables(stmt.Content, resources)

		// Deploy the statement using the global session
		deploymentID, err := fd.deployStatement(ctx, stmt.Name, processedSQL)
		if err != nil {
			return deploymentIDs, fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		deploymentIDs = append(deploymentIDs, deploymentID)
		fd.logger.Info("statement deployed globally", "deployment_id", deploymentID, "name", stmt.Name)

		// Small delay between statements to avoid overwhelming the API
		time.Sleep(500 * time.Millisecond)
	}

	fd.logger.Info("all statements deployed in global session", "count", len(statements))
	return deploymentIDs, nil
}

// deploySessionBased uses the original session-per-pipeline approach
func (fd *FlinkDeployer) deploySessionBased(ctx context.Context, statements []*types.SQLStatement, resources *Resources) ([]string, error) {
	var deploymentIDs []string

	// Create (or retry creating) a session once for all statements using resilient helper
	// Resolve SQL Gateway URL and perform a brief readiness wait (up to 8s) before retries
	gwURL := fd.getSQLGatewayURL()
	readinessCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	if err := fd.waitForSQLGatewayReady(readinessCtx, gwURL, 8*time.Second); err != nil {
		fd.logger.Warn("gateway readiness not confirmed; continuing", "error", err)
	}
	cancel()

	sessID, err := fd.createSessionWithRetry(ctx, "pipegen-session", 7, 2*time.Second)
	if err != nil {
		return nil, err
	}
	fd.sessionID = sessID

	for i, stmt := range statements {
		fd.logger.Info("deploying statement", "index", i+1, "total", len(statements), "name", stmt.Name)

		// Substitute variables in SQL statement
		processedSQL := fd.substituteVariables(stmt.Content, resources)

		// Deploy the statement using the same session
		deploymentID, err := fd.deployStatement(ctx, stmt.Name, processedSQL)
		if err != nil {
			return deploymentIDs, fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		deploymentIDs = append(deploymentIDs, deploymentID)
		fd.logger.Info("statement deployed", "deployment_id", deploymentID, "name", stmt.Name)

		// Small delay between statements to avoid overwhelming the API
		time.Sleep(500 * time.Millisecond)
	}

	fd.logger.Info("all statements deployed successfully", "count", len(statements))
	return deploymentIDs, nil
}

// getOrCreateGlobalSession gets an existing global session or creates a new one
func (fd *FlinkDeployer) getOrCreateGlobalSession(ctx context.Context, sessionName string) (string, error) {
	sqlGatewayURL := fd.getSQLGatewayURL()

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
			fd.logger.Warn("failed to close list response body", "error", err)
		}
	}()

	if listResp.StatusCode == 200 {
		listBody, err := io.ReadAll(listResp.Body)
		if err == nil {
			fd.logger.Debug("current sessions", "body", string(listBody))

			// Try to find existing global session
			if existingSessionID := extractExistingSessionID(string(listBody), sessionName); existingSessionID != "" {
				fd.logger.Info("reusing existing global session", "session_id", existingSessionID)
				return existingSessionID, nil
			}
		}
	}

	// Create new global session
	fd.logger.Info("creating new global session", "name", sessionName)
	sessionEndpoint := fmt.Sprintf("%s/v1/sessions", sqlGatewayURL)
	sessionReqBody := fmt.Sprintf(`{"sessionName": "%s", "properties": {}}`, sessionName)
	sessionReq, err := http.NewRequestWithContext(ctx, "POST", sessionEndpoint, strings.NewReader(sessionReqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create session request: %w", err)
	}
	sessionReq.Header.Set("Content-Type", "application/json")
	sessionResp, err := http.DefaultClient.Do(sessionReq)
	if err != nil {
		fd.logger.Error("global session creation request failed", "error", err)
		return "", fmt.Errorf("failed to create global session: %w", err)
	}
	defer func() {
		if err := sessionResp.Body.Close(); err != nil {
			fd.logger.Warn("failed to close session response body", "error", err)
		}
	}()

	sessionBody, err := io.ReadAll(sessionResp.Body)
	if err != nil {
		fd.logger.Error("failed to read global session response", "error", err)
		return "", fmt.Errorf("failed to read global session response: %w", err)
	}
	fd.logger.Debug("global session creation response", "body", string(sessionBody))
	if sessionResp.StatusCode != 200 {
		fd.logger.Error("global session creation failed", "status", sessionResp.StatusCode, "body", string(sessionBody))
		return "", fmt.Errorf("flink SQL Gateway global session creation failed: %s", string(sessionBody))
	}

	sessionID, perr := parseSessionID(sessionBody)
	if perr != nil {
		fd.logger.Error("could not parse session id from global session response", "error", perr, "body", string(sessionBody))
		return "", fmt.Errorf("could not parse session ID from global session response: %w", perr)
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
	fd.logger.Info("deploying flink sql statements with status tracking", "count", len(statements))

	var deploymentIDs []string

	for i, stmt := range statements {
		fd.logger.Info("deploying statement", "index", i+1, "total", len(statements), "name", stmt.Name)

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
		fd.logger.Info("statement deployed", "deployment_id", deploymentID, "name", stmt.Name)

		// Update status to running
		if statusCallback != nil {
			statusCallback(stmt.Name, "RUNNING", "ACTIVE", deploymentID, "")
		}
	}

	fd.logger.Info("all statements deployed successfully", "count", len(statements))
	return deploymentIDs, nil
}

// deployStatement deploys a single FlinkSQL statement
func (fd *FlinkDeployer) deployStatement(ctx context.Context, name, sql string) (string, error) {
	mode := "session-based"
	if fd.globalMode {
		mode = "global"
	}
	fd.logger.Info("deploying statement", "name", name, "mode", mode)

	// 2. Submit statement to session (reuse sessionID)
	if fd.sessionID == "" {
		return "", fmt.Errorf("no sessionID available for statement deployment")
	}
	sqlGatewayURL := fd.getSQLGatewayURL()
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
			fd.logger.Warn("failed to close statement response body", "error", err)
		}
	}()
	statementBody, err := io.ReadAll(statementResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read statement response: %w", err)
	}
	if statementResp.StatusCode != 200 {
		fd.logger.Error("statement submission failed", "status", statementResp.StatusCode, "body", string(statementBody))
		return "", fmt.Errorf("flink SQL Gateway returned status %d: %s", statementResp.StatusCode, string(statementBody))
	}

	operationHandle, err := parseStatementOperationHandle(statementBody)
	if err != nil {
		return "", fmt.Errorf("failed to parse statement submit response: %w", err)
	}

	// 3. Poll operation status until finished
	opStatusEndpoint := fmt.Sprintf("%s/v1/sessions/%s/operations/%s/status", sqlGatewayURL, fd.sessionID, operationHandle)
	var opStatus string
	var opError string
	for i := 0; i < 30; i++ { // up to 30 attempts, 1s apart
		select {
		case <-ctx.Done():
			fd.logger.Error("context cancelled while waiting for operation status", "name", name)
			return "", fmt.Errorf("context cancelled while waiting for operation status")
		default:
		}
		req, err := http.NewRequestWithContext(ctx, "GET", opStatusEndpoint, nil)
		if err != nil {
			fd.logger.Error("create op status request failed", "name", name, "err", err)
			return "", fmt.Errorf("failed to create operation status request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fd.logger.Warn("poll operation status error", "name", name, "err", err)
			return "", fmt.Errorf("failed to poll operation status: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			fd.logger.Warn("failed to close operation status body", "error", err)
		}
		if err != nil {
			fd.logger.Warn("failed to read operation status response", "name", name, "error", err)
			return "", fmt.Errorf("failed to read operation status response: %w", err)
		}
		statusResp, perr := parseOperationStatus(body)
		if perr != nil {
			fd.logger.Error("parse op status failed", "name", name, "err", perr, "body", string(body))
			return "", perr
		}
		opStatus = statusResp.Status
		opError = statusResp.Error
		if opStatus == "FINISHED" {
			fd.logger.Info("sql statement executed", "name", name)
			return name, nil
		}
		if opStatus == "ERROR" || opError != "" {
			// Attempt to retrieve detailed error via helper (handles /result/0 fallback)
			if details, derr := fd.fetchOperationResult(ctx, sqlGatewayURL, operationHandle); derr == nil && details != "" {
				fd.logger.Error("statement error details", "name", name, "details", details)
			} else if derr != nil {
				fd.logger.Warn("result detail retrieval failed", "name", name, "error", derr)
			}
			fd.logger.Error("statement execution error", "name", name, "error", opError)
			return "", fmt.Errorf("SQL statement '%s' failed: %s", name, opError)
		}
		time.Sleep(1 * time.Second)
	}
	fd.logger.Error("statement did not finish after polling", "name", name, "status", opStatus, "error", opError)
	return "", fmt.Errorf("SQL statement '%s' did not finish after polling. Last status: %s, error: %s", name, opStatus, opError)
}

// (Removed legacy substring-based extractOperation* helpers in favor of typed JSON parsing)

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
				fd.logger.Warn("failed to close result body", "error", cerr)
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

// (Removed legacy extractSessionID; replaced with parseSessionID using typed struct)

// escapeJSONString escapes quotes and newlines for JSON string
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// createSessionWithRetry attempts to create a Flink SQL Gateway session with retries & backoff.
// Rationale: After container startup the gateway port (8083) may accept connections but the
// backend service (Dispatcher / REST endpoint) might still be initializing, causing connection
// resets (ECONNRESET) or 5xx/4xx transient errors. We retry a limited number of times with
// exponential backoff and jitter-less simplicity (deterministic) to keep logs predictable.
func (fd *FlinkDeployer) createSessionWithRetry(ctx context.Context, sessionName string, maxAttempts int, initialBackoff time.Duration) (string, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	sqlGatewayURL := fd.getSQLGatewayURL()
	sessionEndpoint := fmt.Sprintf("%s/v1/sessions", sqlGatewayURL)
	payload := fmt.Sprintf(`{"sessionName": "%s", "properties": {}}`, sessionName)

	var lastErr error
	backoff := initialBackoff
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled while creating session: %w", ctx.Err())
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "POST", sessionEndpoint, strings.NewReader(payload))
		if err != nil {
			return "", fmt.Errorf("failed to build session request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("session creation attempt %d/%d failed: %w", attempt, maxAttempts, err)
			fd.logger.Warn("session creation attempt failed", "attempt", attempt, "max", maxAttempts, "error", err)
		} else {
			body, rerr := io.ReadAll(resp.Body)
			if cerr := resp.Body.Close(); cerr != nil {
				fd.logger.Warn("failed to close session body", "error", cerr)
			}
			if rerr != nil {
				lastErr = fmt.Errorf("failed reading session response (attempt %d): %w", attempt, rerr)
				fd.logger.Warn("failed reading session response", "attempt", attempt, "error", rerr)
			} else if resp.StatusCode == 200 {
				id, perr := parseSessionID(body)
				if perr == nil {
					if attempt > 1 {
						fd.logger.Info("session created after attempts", "attempts", attempt, "session_id", id)
					} else {
						fd.logger.Info("session created", "session_id", id)
					}
					return id, nil
				}
				lastErr = fmt.Errorf("attempt %d: session parse failure: %v body=%s", attempt, perr, string(body))
				fd.logger.Warn("session parse failure", "attempt", attempt, "error", perr, "body", string(body))
			} else {
				lastErr = fmt.Errorf("attempt %d: non-200 status %d: %s", attempt, resp.StatusCode, string(body))
				fd.logger.Warn("non-200 session create status", "attempt", attempt, "status", resp.StatusCode, "body", string(body))
			}
		}

		// Backoff before next attempt if not last
		if attempt < maxAttempts {
			fd.logger.Info("retrying session creation", "next_backoff", backoff.String(), "attempt", attempt, "max", maxAttempts)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return "", fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-timer.C:
			}
			// Exponential backoff cap at 30s
			backoff = backoff * 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unknown failure creating session")
	}
	return "", lastErr
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
	fd.logger.Info("cleaning up flink deployments", "count", len(deploymentIDs))

	for _, deploymentID := range deploymentIDs {
		if err := fd.stopDeployment(ctx, deploymentID); err != nil {
			return fmt.Errorf("failed to stop deployment %s: %w", deploymentID, err)
		}
		fd.logger.Info("stopped deployment", "deployment_id", deploymentID)
	}

	fd.logger.Info("all deployments cleaned up")
	return nil
}

// defaultStopDeployment is the actual implementation
func (fd *FlinkDeployer) defaultStopDeployment(ctx context.Context, deploymentID string) error {
	fd.logger.Info("stopping deployment", "deployment_id", deploymentID)

	// Create a new context with timeout for cleanup operations to avoid cancellation issues
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all running jobs and cancel them (since deploymentID is just a statement name,
	// we can't match by name easily, so we cancel all running jobs)
	jobsEndpoint := fmt.Sprintf("%s/jobs", fd.config.FlinkURL)
	jobsResp, err := http.Get(jobsEndpoint)
	if err != nil {
		fd.logger.Warn("failed to get job list for cleanup", "error", err)
		return nil // Don't fail completely, just warn
	}
	defer func() { _ = jobsResp.Body.Close() }()

	if jobsResp.StatusCode != http.StatusOK {
		fd.logger.Warn("failed to get job list", "status", jobsResp.StatusCode)
		return nil
	}

	jobsBody, err := io.ReadAll(jobsResp.Body)
	if err != nil {
		fd.logger.Warn("failed to read jobs response", "error", err)
		return nil
	}

	var jobsResponse struct {
		Jobs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(jobsBody, &jobsResponse); err != nil {
		fd.logger.Warn("failed to parse jobs response", "error", err)
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
				fd.logger.Warn("failed to create cancel request", "job_id", job.ID, "error", err)
				continue
			}

			cancelResp, err := http.DefaultClient.Do(cancelReq)
			if err != nil {
				fd.logger.Warn("failed to cancel job", "job_id", job.ID, "error", err)
				continue
			}
			_ = cancelResp.Body.Close()

			if cancelResp.StatusCode == http.StatusAccepted {
				cancelledJobs = append(cancelledJobs, job.ID)
				fd.logger.Info("cancelled job", "job_id", job.ID)
			} else {
				fd.logger.Warn("failed to cancel job", "job_id", job.ID, "status", cancelResp.StatusCode)
			}
		}
	}

	// Wait a moment for jobs to be cancelled
	if len(cancelledJobs) > 0 {
		fd.logger.Info("waiting for jobs to cancel", "count", len(cancelledJobs))
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
	logger := logpkg.Global()
	logger.Info("cancelling all running flink jobs")

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
				logger.Warn("failed to create cancel request", "job_id", job.ID, "error", err)
				continue
			}

			cancelResp, err := http.DefaultClient.Do(cancelReq)
			if err != nil {
				logger.Warn("failed to cancel job", "job_id", job.ID, "error", err)
				continue
			}
			_ = cancelResp.Body.Close()

			if cancelResp.StatusCode == http.StatusAccepted {
				cancelledCount++
				logger.Info("cancelled job", "job_id", job.ID)
			} else {
				logger.Warn("failed to cancel job", "job_id", job.ID, "status", cancelResp.StatusCode)
			}
		}
	}

	if cancelledCount > 0 {
		logger.Info("waiting for jobs to cancel", "count", cancelledCount)
		time.Sleep(3 * time.Second)
		logger.Info("cancelled flink jobs", "count", cancelledCount)
	} else {
		logger.Info("no running jobs to cancel")
	}

	return nil
}
