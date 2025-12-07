package xai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://api.x.ai/v1"
)

const (
	ModelGrokImagineV0p9        Model = "grok-imagine-v0p9"
	ModelGrok2Image             Model = "grok-2-image"
	ModelGrok41Fast             Model = "grok-4-1-fast"
	ModelGrok41FastNonReasoning Model = "grok-4-1-fast-non-reasoning"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type Model string

type ChatRequest struct {
	Messages []Message `json:"messages"`
	Model    Model     `json:"model"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

func (c *Client) CreateChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = ModelGrok41Fast // Default model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("xai api error: status=%d body=%s", resp.StatusCode, errorBody.String())
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

type ImageRequest struct {
	Prompt         string `json:"prompt"`
	Model          string `json:"model"`
	ValidationMode string `json:"validation_mode,omitempty"`
}

type ImageResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

type ImageData struct {
	Url string `json:"url"`
}

// GenerateImage creates an image using the ModelGrokImagineV0p9 model.
// It returns the content of the generated image response (typically an image URL).
func (c *Client) GenerateImage(ctx context.Context, prompt string) (string, error) {
	req := ImageRequest{
		Model:  string(ModelGrokImagineV0p9),
		Prompt: prompt,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/images/generations", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return "", fmt.Errorf("xai api error: status=%d body=%s", resp.StatusCode, errorBody.String())
	}

	var imgResp ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", err
	}

	if len(imgResp.Data) == 0 {
		return "", fmt.Errorf("no data returned from image generation")
	}

	return imgResp.Data[0].Url, nil
}

type ToolType string

const (
	ToolTypeWebSearch     ToolType = "web_search"
	ToolTypeXSearch       ToolType = "x_search"
	ToolTypeCodeExecution ToolType = "code_execution"
	ToolTypeCollections   ToolType = "collections_search"
	ToolTypeViewXVideo    ToolType = "view_x_video"
	ToolTypeViewImage     ToolType = "view_image"
)

type ResponseRequest struct {
	Model  string         `json:"model"`
	Input  []Message      `json:"input"`
	Tools  []ResponseTool `json:"tools,omitempty"`
	Stream bool           `json:"stream,omitempty"`
}

type ResponseTool struct {
	Type                     ToolType          `json:"type"`
	Filters                  *WebSearchFilters `json:"filters,omitempty"`
	EnableImageUnderstanding bool              `json:"enable_image_understanding,omitempty"`
}

type WebSearchFilters struct {
	ExcludedDomains []string `json:"excluded_domains,omitempty"`
}

type ResponsesResponse struct {
	Output    []ResponseItem `json:"output"`
	Citations []string       `json:"citations"`
	Usage     interface{}    `json:"usage"` // Simplified for now
}

type ResponseItem struct {
	Type     string      `json:"type"` // "text", "tool_call", etc.
	Content  interface{} `json:"content,omitempty"`
	ToolCall *ToolCall   `json:"tool_call,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// GenerateResponse calls the /v1/responses endpoint, supporting tools like web_search.
// This endpoint returns a ResponsesResponse with an Output array.
func (c *Client) GenerateResponse(ctx context.Context, req ResponseRequest) (*ResponsesResponse, error) {
	if req.Model == "" {
		req.Model = string(ModelGrok41Fast)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("xai api error: status=%d body=%s", resp.StatusCode, errorBody.String())
	}

	var responsesResp ResponsesResponse

	// Read body to bytes to allow re-reading/logging
	bodyBytes, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(bodyBytes, &responsesResp); err != nil {
		// Log raw body for debugging
		return nil, fmt.Errorf("failed to unmarshal response: %v. Raw body: %s", err, string(bodyBytes))
	}

	if len(responsesResp.Output) == 0 {
		return nil, fmt.Errorf("unexpected response structure (no output): %s", string(bodyBytes))
	}

	return &responsesResp, nil
}
