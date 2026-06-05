package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	Debug      bool
	LogFile    string
	Directory  string
	mu         sync.Mutex
}

type HealthResponse struct {
	Healthy bool   `json:"healthy"`
	Version string `json:"version"`
}

type sessionResponse struct {
	ID string `json:"id"`
}

type sendMessageRequest struct {
	Parts []messagePart `json:"parts"`
}

type messagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MessageInfo struct {
	Role  string `json:"role"`
	Model string `json:"model"`
}

type sendMessageResponse struct {
	Info  MessageInfo    `json:"info"`
	Parts []responsePart `json:"parts"`
}

type responsePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ProvidersResponse struct {
	Providers []map[string]interface{} `json:"providers"`
	Default   map[string]string        `json:"default"`
}

func (c *Client) readBody(req *http.Request, resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	c.logResp(req, resp, body)
	return body, nil
}

func (c *Client) logResp(req *http.Request, resp *http.Response, body []byte) {
	if !c.Debug {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	f, err := os.OpenFile(c.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n[API] %s %s\n", req.Method, req.URL)
	fmt.Fprintf(f, "  Status: %d\n", resp.StatusCode)
	fmt.Fprintf(f, "  Body: %s\n\n", string(body))
}

type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type QuestionData struct {
	Question string           `json:"question"`
	Header   string           `json:"header"`
	Options  []QuestionOption `json:"options"`
	Multiple bool             `json:"multiple,omitempty"`
}

type SSEMessage struct {
	Payload SSEMessagePayload `json:"payload"`
}

type SSEMessagePayload struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

type QuestionProperties struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionID"`
	Questions []QuestionData `json:"questions"`
}

type PermissionReqInfo struct {
	ID         string   `json:"id"`
	SessionID  string   `json:"sessionID"`
	Permission string   `json:"permission"`
	Patterns   []string `json:"patterns"`
}

type ControlRequest struct {
	ID   string             `json:"id"`
	Type string             `json:"type"`
	Data ControlRequestData `json:"data"`
}

type ControlRequestData struct {
	Questions []QuestionData `json:"questions"`
}

func (c *Client) SubscribeGlobalEvents(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/global/event", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.Directory != "" {
		req.Header.Set("x-opencode-directory", c.Directory)
	}

	httpClient := &http.Client{
		Transport: c.httpClient.Transport,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("subscribe events: unexpected status %d", resp.StatusCode)
	}
	return resp, nil
}

func (c *Client) ReplyToPermission(id string, reply string) error {
	b, _ := json.Marshal(map[string]string{"reply": reply})
	req, err := http.NewRequest("POST", c.baseURL+"/permission/"+id+"/reply", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Directory != "" {
		req.Header.Set("x-opencode-directory", c.Directory)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reply to permission: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ReplyToQuestion(id string, answers [][]string) error {
	b, _ := json.Marshal(map[string][][]string{"answers": answers})
	req, err := http.NewRequest("POST", c.baseURL+"/question/"+id+"/reply", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Directory != "" {
		req.Header.Set("x-opencode-directory", c.Directory)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reply to question: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func New(baseURL string) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
	}
	return &Client{baseURL: baseURL, httpClient: &http.Client{Transport: transport}}
}

func (c *Client) GetProviders() (*ProvidersResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/config/providers", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := c.readBody(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get providers: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var result ProvidersResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPath() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/path", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := c.readBody(req, resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get path: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var result map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return "", err
	}
	for _, key := range []string{"directory", "worktree", "path"} {
		if p, ok := result[key]; ok {
			if s, ok := p.(string); ok {
				return s, nil
			}
		}
	}
	return "", nil
}

func (c *Client) GetSession(id string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/session/"+id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := c.readBody(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get session: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var result map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateSession(title string) (string, error) {
	body, _ := json.Marshal(map[string]string{"title": title})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	b, err := c.readBody(req, resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create session: unexpected status %d: %s", resp.StatusCode, string(b))
	}

	var s sessionResponse
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&s); err != nil {
		return "", err
	}
	return s.ID, nil
}

func (c *Client) SendMessage(sessionID, text string) (string, string, error) {
	body, _ := json.Marshal(sendMessageRequest{
		Parts: []messagePart{{Type: "text", Text: text}},
	})
	req, err := http.NewRequest("POST", c.baseURL+"/session/"+sessionID+"/message", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	b, err := c.readBody(req, resp)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("send message: unexpected status %d: %s", resp.StatusCode, string(b))
	}

	var msg sendMessageResponse
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&msg); err != nil {
		return "", "", err
	}

	for _, p := range msg.Parts {
		if p.Type == "text" {
			return p.Text, msg.Info.Model, nil
		}
	}
	return "", "", fmt.Errorf("no text part in response")
}

func (c *Client) SendMessageRaw(sessionID, text string) (*http.Response, error) {
	body, _ := json.Marshal(sendMessageRequest{
		Parts: []messagePart{{Type: "text", Text: text}},
	})
	req, err := http.NewRequest("POST", c.baseURL+"/session/"+sessionID+"/message", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) Health() (*HealthResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/global/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	b, err := c.readBody(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, string(b))
	}

	var result HealthResponse
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
