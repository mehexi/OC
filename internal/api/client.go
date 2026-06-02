package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
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

func (c *Client) GetProviders() (*ProvidersResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/config/providers", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get providers: unexpected status %d", resp.StatusCode)
	}
	var result ProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPath() (string, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/path", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get path: unexpected status %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if p, ok := result["path"]; ok {
		if s, ok := p.(string); ok {
			return s, nil
		}
	}
	return "", nil
}

func (c *Client) GetSession(id string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/session/"+id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get session: unexpected status %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpClient: http.DefaultClient}
}

func (c *Client) CreateSession(title string) (string, error) {
	body, _ := json.Marshal(map[string]string{"title": title})
	req, err := http.NewRequest("POST", c.baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create session: unexpected status %d", resp.StatusCode)
	}

	var s sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("send message: unexpected status %d", resp.StatusCode)
	}

	var msg sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return "", "", err
	}

	for _, p := range msg.Parts {
		if p.Type == "text" {
			return p.Text, msg.Info.Model, nil
		}
	}
	return "", "", fmt.Errorf("no text part in response")
}

func (c *Client) Health() (*HealthResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/global/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
