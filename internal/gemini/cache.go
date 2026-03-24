package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CacheManager manages Gemini context caches via the /v1beta/cachedContents
// REST resource. Context caching is a stateful resource: content is uploaded
// once and receives a cacheName URI that is referenced in subsequent
// generateContent requests for a significant cost discount.
type CacheManager struct {
	client *Client
}

// NewCacheManager creates a CacheManager that shares the Client's HTTP
// transport and authentication.
func NewCacheManager(client *Client) *CacheManager {
	return &CacheManager{client: client}
}

// CacheEntry represents a cached content resource returned by the API.
type CacheEntry struct {
	Name          string         `json:"name"`
	Model         string         `json:"model"`
	DisplayName   string         `json:"displayName"`
	CreateTime    string         `json:"createTime"`
	UpdateTime    string         `json:"updateTime"`
	ExpireTime    string         `json:"expireTime"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
}

// cacheListResponse is the JSON envelope for the list endpoint.
type cacheListResponse struct {
	CachedContents []CacheEntry `json:"cachedContents"`
	NextPageToken  string       `json:"nextPageToken,omitempty"`
}

// cacheEndpoint returns the base URL for the cachedContents resource.
func (cm *CacheManager) cacheEndpoint() string {
	return fmt.Sprintf("%s/%s/cachedContents", cm.client.baseURL, cm.client.apiVersion)
}

// Create uploads stable content (system instruction, tools, conversation
// prefix) and returns a CacheEntry whose Name field is used in subsequent
// generateContent requests.
//
// The model parameter must be specified as a full resource path:
// "models/gemini-2.5-flash" -- the method prepends "models/" if needed.
//
// ttl is a Go duration converted to the Gemini "Ns" format. If zero the API
// default is used. Set either ttl OR expireTime (via CachedContentRequest
// directly for expire-time based expiry).
func (cm *CacheManager) Create(ctx context.Context, req *CachedContentRequest) (*CacheEntry, error) {
	if req == nil {
		return nil, fmt.Errorf("cached content request must not be nil")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("model is required for cache creation")
	}

	// Ensure models/ prefix is present (API requires full resource path).
	if stripModelPrefix(req.Model) == req.Model {
		// No "models/" prefix found -- add it.
		req.Model = "models/" + req.Model
	}

	// Validate minimum token threshold.
	if err := cm.validateMinTokens(req.Model, req.Contents, req.SystemInstruction, req.Tools); err != nil {
		return nil, err
	}

	url := cm.cacheEndpoint()
	httpReq, err := cm.client.buildRequest(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, fmt.Errorf("build cache create request: %w", err)
	}

	resp, err := cm.client.doRequest(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("cache create: %w", err)
	}
	defer resp.Body.Close()

	var entry CacheEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode cache create response: %w", err)
	}
	return &entry, nil
}

// Get retrieves a cached content resource by its name (e.g. "cachedContents/abc123").
func (cm *CacheManager) Get(ctx context.Context, name string) (*CacheEntry, error) {
	if name == "" {
		return nil, fmt.Errorf("cache name must not be empty")
	}

	url := fmt.Sprintf("%s/%s/%s", cm.client.baseURL, cm.client.apiVersion, name)
	req, err := cm.client.buildRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build cache get request: %w", err)
	}

	resp, err := cm.client.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}
	defer resp.Body.Close()

	var entry CacheEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode cache get response: %w", err)
	}
	return &entry, nil
}

// List returns all cached content entries, handling pagination automatically.
func (cm *CacheManager) List(ctx context.Context) ([]CacheEntry, error) {
	var all []CacheEntry
	baseURL := cm.cacheEndpoint()

	pageToken := ""
	for {
		url := baseURL
		if pageToken != "" {
			url += "?pageToken=" + pageToken
		}

		req, err := cm.client.buildRequest(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build cache list request: %w", err)
		}

		resp, err := cm.client.doRequest(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("cache list: %w", err)
		}

		var listResp cacheListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode cache list response: %w", err)
		}
		resp.Body.Close()

		all = append(all, listResp.CachedContents...)

		if listResp.NextPageToken == "" {
			break
		}
		pageToken = listResp.NextPageToken
	}

	return all, nil
}

// Delete removes a cached content resource.
func (cm *CacheManager) Delete(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("cache name must not be empty")
	}

	url := fmt.Sprintf("%s/%s/%s", cm.client.baseURL, cm.client.apiVersion, name)
	req, err := cm.client.buildRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("build cache delete request: %w", err)
	}

	resp, err := cm.client.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

// cacheUpdateRequest is the PATCH body for updating a cache's TTL.
type cacheUpdateRequest struct {
	TTL        string `json:"ttl,omitempty"`
	ExpireTime string `json:"expireTime,omitempty"`
}

// Update extends or changes the TTL of an existing cache.
// The updateMask query parameter is required by the Gemini REST API for PATCH.
func (cm *CacheManager) Update(ctx context.Context, name string, ttl time.Duration) (*CacheEntry, error) {
	if name == "" {
		return nil, fmt.Errorf("cache name must not be empty")
	}

	url := fmt.Sprintf("%s/%s/%s?updateMask=ttl", cm.client.baseURL, cm.client.apiVersion, name)
	body := cacheUpdateRequest{
		TTL: formatTTL(ttl),
	}
	req, err := cm.client.buildRequest(ctx, http.MethodPatch, url, body)
	if err != nil {
		return nil, fmt.Errorf("build cache update request: %w", err)
	}

	resp, err := cm.client.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cache update: %w", err)
	}
	defer resp.Body.Close()

	var entry CacheEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode cache update response: %w", err)
	}
	return &entry, nil
}

// UseCachedContent sets the cachedContent field on a GenerateContentRequest
// so the API reuses the cached context instead of re-processing it.
func UseCachedContent(request *GenerateContentRequest, cacheName string) {
	request.CachedContent = cacheName
}

// CacheEconomics calculates the cost comparison between using cached content
// and sending everything fresh. Returns cost with cache, cost without, and
// the break-even number of requests.
type CacheCostComparison struct {
	CostWithCache    float64 `json:"costWithCache"`
	CostWithoutCache float64 `json:"costWithoutCache"`
	SavingsPerReq    float64 `json:"savingsPerRequest"`
	BreakEvenReqs    int     `json:"breakEvenRequests"`
}

// CacheEconomics calculates the cost comparison for caching vs no caching.
// inputTokens is the total input tokens, cachedTokens is how many are cached.
//
// The break-even calculation estimates how many requests are needed before
// cumulative cache storage costs are recovered by per-request savings.
// Cache storage cost is approximately the cached input price per hour
// (API charges for keeping the cache alive).
func CacheEconomics(modelID string, inputTokens, cachedTokens, outputTokens int) (CacheCostComparison, error) {
	withCache, err := TokenEconomics(modelID, inputTokens, outputTokens, cachedTokens)
	if err != nil {
		return CacheCostComparison{}, fmt.Errorf("calculate cached cost: %w", err)
	}
	withoutCache, err := TokenEconomics(modelID, inputTokens, outputTokens, 0)
	if err != nil {
		return CacheCostComparison{}, fmt.Errorf("calculate uncached cost: %w", err)
	}

	savings := withoutCache.TotalCost - withCache.TotalCost

	// Break-even: estimate how many requests are needed for savings to
	// exceed the one-time cache creation overhead.
	// Cache creation cost ~ the full input cost of uploading the content.
	// Storage cost ~ 25% of input price per hour (approximation).
	// With the 75% discount, savings per request are substantial.
	breakEven := 1
	if savings > 0 {
		// Creation overhead: one full-price input call for the cached tokens.
		model, modelErr := GetModel(stripModelPrefix(modelID))
		if modelErr == nil && model.InputPricePerM > 0 {
			creationCost := float64(cachedTokens) / 1_000_000.0 * model.InputPricePerM
			breakEven = int(creationCost/savings) + 1
			if breakEven < 1 {
				breakEven = 1
			}
		}
	} else {
		breakEven = 0 // No savings -- caching not beneficial
	}

	return CacheCostComparison{
		CostWithCache:    withCache.TotalCost,
		CostWithoutCache: withoutCache.TotalCost,
		SavingsPerReq:    savings,
		BreakEvenReqs:    breakEven,
	}, nil
}

// SplitContext separates stable content (system prompt, tools -- suitable for
// caching with a long TTL) from active content (conversation history, which
// changes every turn and should NOT be cached).
//
// Returns stableMessages containing messages at the start of the conversation
// that are repeated (system context, initial setup), and activeMessages
// containing the dynamic conversation tail.
//
// Heuristic: all messages except the last N user/model pairs are considered
// stable. If there are fewer than stableThreshold messages, everything is
// active (not worth caching).
func SplitContext(messages []Message, stableThreshold int) (stable, active []Message) {
	if stableThreshold <= 0 {
		stableThreshold = 4 // Default: keep last 4 messages active
	}

	if len(messages) <= stableThreshold {
		// Not enough history to benefit from caching.
		return nil, messages
	}

	splitPoint := len(messages) - stableThreshold
	stable = make([]Message, splitPoint)
	copy(stable, messages[:splitPoint])
	active = make([]Message, stableThreshold)
	copy(active, messages[splitPoint:])
	return stable, active
}

// validateMinTokens checks that the content to be cached exceeds the model's
// minimum token threshold. This is a heuristic check based on content length
// rather than a precise token count (which would require an API call).
func (cm *CacheManager) validateMinTokens(modelID string, contents []Message, systemInstruction *Message, tools []Tool) error {
	model, err := GetModel(stripModelPrefix(modelID))
	if err != nil {
		// Unknown model -- skip validation rather than block.
		return nil
	}
	if model.MinCacheTokens == 0 {
		return nil
	}

	// Rough estimate: 4 chars per token (English average). This heuristic is
	// unreliable for non-ASCII content (Arabic, CJK) and dense JSON, which
	// tokenise significantly denser. For precise validation, use the
	// CountTokens API before creating a cache.
	var charCount int
	for _, msg := range contents {
		for _, part := range msg.Parts {
			charCount += len(part.Text)
		}
	}
	if systemInstruction != nil {
		for _, part := range systemInstruction.Parts {
			charCount += len(part.Text)
		}
	}

	estimatedTokens := charCount / 4
	if estimatedTokens < model.MinCacheTokens {
		return fmt.Errorf("estimated %d tokens, but model %s requires minimum %d tokens for caching", estimatedTokens, model.ID, model.MinCacheTokens)
	}
	return nil
}

// stripModelPrefix removes the "models/" prefix if present.
// The > check (not >=) is intentional: "models/" alone would yield an empty
// string which is not a valid model ID, so it is left unchanged.
func stripModelPrefix(modelID string) string {
	const prefix = "models/"
	if len(modelID) > len(prefix) && modelID[:len(prefix)] == prefix {
		return modelID[len(prefix):]
	}
	return modelID
}

// formatTTL converts a Go duration to the Gemini TTL format ("Ns").
func formatTTL(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
