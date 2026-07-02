package pilotdeck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// AgentRequest is the JSON body sent to POST /api/agent.
type AgentRequest struct {
	ProjectPath string `json:"projectPath"`
	Message     string `json:"message"`
	Stream      bool   `json:"stream"`
	SessionID   string `json:"sessionId,omitempty"`
}

// AgentResponse is the JSON body returned by POST /api/agent.
type AgentResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"sessionId"`
	Error     string `json:"error,omitempty"`
	// RawResponse is the full JSON response body, populated after unmarshal
	// for logging purposes (not part of the API contract).
	RawResponse []byte `json:"-"`
}

// SubmitMessage sends a message to the PilotDeck agent API.
// sessionID may be "" to start a new conversation.
func SubmitMessage(ctx context.Context, baseURL, apiKey, projectPath, message, sessionID string) (*AgentResponse, error) {
	reqBody := AgentRequest{
		ProjectPath: projectPath,
		Message:     message,
		Stream:      false,
		SessionID:   sessionID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := baseURL + "/api/agent"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("x-api-key", apiKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	var agentResp AgentResponse
	if err := json.Unmarshal(respBody, &agentResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	agentResp.RawResponse = respBody
	if !agentResp.Success {
		return &agentResp, fmt.Errorf("api error: %s", agentResp.Error)
	}

	return &agentResp, nil
}

// HealthCheck verifies the PilotDeck server is reachable via GET /health.
func HealthCheck(ctx context.Context, baseURL string) error {
	url := baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// FetchSessionMessages retrieves the full conversation messages for a session.
func FetchSessionMessages(ctx context.Context, baseURL, apiKey, sessionID, projectPath string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/sessions/%s/messages?projectPath=%s",
		baseURL, url.QueryEscape(sessionID), url.QueryEscape(projectPath))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create messages request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read messages: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch messages: http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
