package gemini

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
	// DefaultBaseURL is the Gemini API base URL.
	DefaultBaseURL = "https://generativelanguage.googleapis.com"
	// DefaultAPIVersion is the API version prefix. Use v1beta for all
	// features (context caching, grounding, 3.x models).
	DefaultAPIVersion = "v1beta"
)

// Client is the Gemini REST API client.
type Client struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	apiVersion  string
	defaultModel string
	genConfig   *GenerationConfig
	safety      []SafetySetting
	retryConfig *RetryConfig
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) { c.apiKey = key }
}

// WithModel sets the default model for requests.
func WithModel(model string) ClientOption {
	return func(c *Client) { c.defaultModel = model }
}

// WithBaseURL sets the API base URL (without version prefix).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

// WithAPIVersion sets the API version (e.g., "v1beta", "v1").
func WithAPIVersion(version string) ClientOption {
	return func(c *Client) { c.apiVersion = version }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = client }
}

// WithGenerationConfig sets the default generation configuration.
func WithGenerationConfig(cfg GenerationConfig) ClientOption {
	return func(c *Client) { c.genConfig = &cfg }
}

// WithSafetySettings sets the default safety settings.
func WithSafetySettings(settings []SafetySetting) ClientOption {
	return func(c *Client) { c.safety = settings }
}

// WithRetryConfig enables retry with the given configuration.
func WithRetryConfig(cfg RetryConfig) ClientOption {
	return func(c *Client) { c.retryConfig = &cfg }
}

// NewClient creates a new Gemini API client with the given options.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		baseURL:    DefaultBaseURL,
		apiVersion: DefaultAPIVersion,
		defaultModel: "gemini-2.5-flash",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ModelEndpoint returns the full URL for a model endpoint.
func (c *Client) ModelEndpoint(model, action string) string {
	if model == "" {
		model = c.defaultModel
	}
	return fmt.Sprintf("%s/%s/models/%s:%s", c.baseURL, c.apiVersion, model, action)
}

// buildRequest constructs an HTTP request with authentication and content type.
func (c *Client) buildRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-goog-api-key", c.apiKey)
	}

	return req, nil
}

// doRequest executes an HTTP request with optional retry, returning the response.
// Caller is responsible for closing the response body.
func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.retryConfig != nil {
		return ExecuteWithRetry(ctx, *c.retryConfig, func(ctx context.Context) (*http.Response, error) {
			// Clone request for retry. req.GetBody is auto-populated by
			// http.NewRequestWithContext when body is a *bytes.Reader,
			// so Clone() safely duplicates the body without manual io.ReadAll.
			clonedReq := req.Clone(ctx)
			return c.executeAndCheck(clonedReq)
		})
	}
	return c.executeAndCheck(req)
}

// executeAndCheck performs the HTTP request and converts non-2xx to APIError.
func (c *Client) executeAndCheck(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return resp, nil
}

// GenerateContent sends a synchronous (non-streaming) request to the Gemini API.
func (c *Client) GenerateContent(ctx context.Context, model string, request *GenerateContentRequest) (*GenerateContentResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request must not be nil")
	}
	if model == "" {
		model = c.defaultModel
	}

	// Apply defaults.
	if request.GenerationConfig == nil && c.genConfig != nil {
		request.GenerationConfig = c.genConfig
	}
	if len(request.SafetySettings) == 0 && len(c.safety) > 0 {
		request.SafetySettings = c.safety
	}

	url := c.ModelEndpoint(model, "generateContent")
	req, err := c.buildRequest(ctx, http.MethodPost, url, request)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CountTokens counts the tokens in the given content.
func (c *Client) CountTokens(ctx context.Context, model string, request *CountTokensRequest) (*CountTokensResponse, error) {
	if model == "" {
		model = c.defaultModel
	}

	url := c.ModelEndpoint(model, "countTokens")
	req, err := c.buildRequest(ctx, http.MethodPost, url, request)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CountTokensResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// StreamEndpoint returns the SSE streaming URL for generateContent.
func (c *Client) StreamEndpoint(model string) string {
	if model == "" {
		model = c.defaultModel
	}
	return fmt.Sprintf("%s/%s/models/%s:streamGenerateContent?alt=sse", c.baseURL, c.apiVersion, model)
}

// buildStreamRequest creates a request for SSE streaming.
// The actual streaming parser is in streaming.go.
func (c *Client) buildStreamRequest(ctx context.Context, model string, request *GenerateContentRequest) (*http.Request, error) {
	if request == nil {
		return nil, fmt.Errorf("request must not be nil")
	}
	if model == "" {
		model = c.defaultModel
	}

	// Apply defaults.
	if request.GenerationConfig == nil && c.genConfig != nil {
		request.GenerationConfig = c.genConfig
	}
	if len(request.SafetySettings) == 0 && len(c.safety) > 0 {
		request.SafetySettings = c.safety
	}

	url := c.StreamEndpoint(model)
	return c.buildRequest(ctx, http.MethodPost, url, request)
}
