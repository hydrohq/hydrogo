package ai

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const encryptedKeyPrefix = "enc:v1:"

// ChatMessage represents a single turn in a multi-adapter AI conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest describes a provider-neutral generation request.
type ChatRequest struct {
	SystemPrompt string        `json:"system_prompt,omitempty"`
	Messages     []ChatMessage `json:"messages"`
	JSONMode     bool          `json:"json_mode,omitempty"`
	MaxTokens    int           `json:"max_tokens,omitempty"`
}

// ChatResponse returns the model's output along with token metering metadata.
type ChatResponse struct {
	Content    string `json:"content"`
	TokensUsed int64  `json:"tokens_used"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
}

// Adapter is the unified multi-provider AI interface (Gemini, OpenAI, Anthropic, OpenAI-Compatible).
type Adapter interface {
	Generate(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// AdapterConfig holds the runtime configuration for instantiating a tenant or platform AI adapter.
type AdapterConfig struct {
	Provider string
	Model    string
	APIKey   string
	Endpoint string
	Client   *http.Client
}

// NewAdapter constructs the appropriate AI provider adapter for the given configuration.
func NewAdapter(cfg AdapterConfig) Adapter {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	model := strings.TrimSpace(cfg.Model)
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	// Custom endpoints (HYDRO_AI_BASE_URL) pass the same SSRF gate on every
	// adapter, not only the OpenAI-compatible one.
	if raw := strings.TrimSpace(cfg.Endpoint); raw != "" {
		if _, err := endpointSecurityCheck(raw); err != nil {
			return rejectedAdapter{err: err}
		}
	}
	switch provider {
	case "gemini", "google":
		if model == "" {
			model = "gemini-2.5-flash"
		}
		return GeminiAdapter{
			APIKey:   strings.TrimSpace(cfg.APIKey),
			Model:    model,
			Endpoint: strings.TrimSpace(cfg.Endpoint),
			Client:   client,
		}
	case "anthropic", "claude":
		if model == "" {
			model = "claude-3-5-sonnet-latest"
		}
		return AnthropicAdapter{
			APIKey:   strings.TrimSpace(cfg.APIKey),
			Model:    model,
			Endpoint: strings.TrimSpace(cfg.Endpoint),
			Client:   client,
		}
	case "openai", "openai_compatible", "ollama", "groq", "deepseek":
		if model == "" {
			model = "gpt-4o-mini"
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint == "" {
			endpoint = "https://api.openai.com/v1/chat/completions"
		}
		return OpenAIAdapter{
			APIKey:   strings.TrimSpace(cfg.APIKey),
			Model:    model,
			Endpoint: endpoint,
			Client:   client,
		}
	default:
		return GeminiAdapter{
			APIKey: strings.TrimSpace(cfg.APIKey),
			Model:  "gemini-2.5-flash",
			Client: client,
		}
	}
}

// rejectedAdapter fails every call with the configuration error that made the
// endpoint unusable, instead of silently falling back to a default provider.
type rejectedAdapter struct {
	err error
}

func (r rejectedAdapter) Generate(context.Context, ChatRequest) (ChatResponse, error) {
	return ChatResponse{}, r.err
}

// ErrEncryptionNotConfigured means no master secret is available to derive the
// BYOK key from; encrypting (or decrypting) with an implicit default would put
// every deployment on the same key, so the operation is refused instead.
var ErrEncryptionNotConfigured = errors.New("AI key encryption is not configured; set HYDRO_AI_KEY_SECRET")

// EncryptAPIKey encrypts a tenant's BYOK API key using AES-256-GCM derived from masterSecret.
func EncryptAPIKey(plaintext, masterSecret string) (string, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return "", nil
	}
	if strings.HasPrefix(plaintext, encryptedKeyPrefix) {
		return plaintext, nil
	}
	key, err := deriveAESKey(masterSecret)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedKeyPrefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// DecryptAPIKey decrypts an AES-256-GCM encrypted BYOK API key.
func DecryptAPIKey(ciphertext, masterSecret string) (string, error) {
	ciphertext = strings.TrimSpace(ciphertext)
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, encryptedKeyPrefix) {
		return ciphertext, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedKeyPrefix))
	if err != nil {
		return "", err
	}
	key, err := deriveAESKey(masterSecret)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted API key")
	}
	nonce, data := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deriveAESKey(secret string) ([32]byte, error) {
	if strings.TrimSpace(secret) == "" {
		return [32]byte{}, ErrEncryptionNotConfigured
	}
	return sha256.Sum256([]byte(secret)), nil
}

// GeminiAdapter implements Adapter for Google Gemini models via the REST v1beta API.
type GeminiAdapter struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

func (g GeminiAdapter) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(g.APIKey) == "" {
		return ChatResponse{}, ErrProvider
	}
	model := strings.TrimSpace(g.Model)
	if model == "" {
		model = "gemini-2.5-flash"
	}
	baseURL := strings.TrimRight(strings.TrimSpace(g.Endpoint), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, model, g.APIKey)

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}
	var contents []content
	for _, m := range req.Messages {
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		contents = append(contents, content{
			Role:  role,
			Parts: []part{{Text: m.Content}},
		})
	}

	body := map[string]any{
		"contents": contents,
	}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		body["systemInstruction"] = content{Parts: []part{{Text: req.SystemPrompt}}}
	}
	genCfg := map[string]any{}
	if req.JSONMode {
		genCfg["responseMimeType"] = "application/json"
	}
	if req.MaxTokens > 0 {
		genCfg["maxOutputTokens"] = req.MaxTokens
	}
	if len(genCfg) > 0 {
		body["generationConfig"] = genCfg
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, ErrProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, ErrProvider
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int64 `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 128<<10)).Decode(&parsed); err != nil {
		return ChatResponse{}, ErrProvider
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return ChatResponse{}, ErrProvider
	}
	text := parsed.Candidates[0].Content.Parts[0].Text
	tokens := parsed.UsageMetadata.TotalTokenCount
	if tokens <= 0 {
		tokens = estimateTokens(req.SystemPrompt, text)
	}
	return ChatResponse{
		Content:    text,
		TokensUsed: tokens,
		Provider:   "gemini",
		Model:      model,
	}, nil
}

// OpenAIAdapter implements Adapter for OpenAI and OpenAI-compatible endpoints (Ollama, Groq, DeepSeek).
type OpenAIAdapter struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

func (o OpenAIAdapter) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(o.APIKey) == "" && !strings.Contains(o.Endpoint, "localhost") && !strings.Contains(o.Endpoint, "127.0.0.1") {
		return ChatResponse{}, ErrProvider
	}
	model := strings.TrimSpace(o.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	endpoint := strings.TrimSpace(o.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint = strings.TrimRight(endpoint, "/") + "/chat/completions"
	}

	var msgs []ChatMessage
	if strings.TrimSpace(req.SystemPrompt) != "" {
		msgs = append(msgs, ChatMessage{Role: "system", Content: req.SystemPrompt})
	}
	msgs = append(msgs, req.Messages...)

	body := map[string]any{
		"model":    model,
		"messages": msgs,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	if strings.TrimSpace(o.APIKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, ErrProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, ErrProvider
	}

	var parsed struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 128<<10)).Decode(&parsed); err != nil || len(parsed.Choices) == 0 {
		return ChatResponse{}, ErrProvider
	}
	text := parsed.Choices[0].Message.Content
	tokens := parsed.Usage.TotalTokens
	if tokens <= 0 {
		tokens = estimateTokens(req.SystemPrompt, text)
	}
	return ChatResponse{
		Content:    text,
		TokensUsed: tokens,
		Provider:   "openai",
		Model:      model,
	}, nil
}

// AnthropicAdapter implements Adapter for Anthropic Claude models via /v1/messages.
type AnthropicAdapter struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

func (a AnthropicAdapter) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(a.APIKey) == "" {
		return ChatResponse{}, ErrProvider
	}
	model := strings.TrimSpace(a.Model)
	if model == "" {
		model = "claude-3-5-sonnet-latest"
	}
	endpoint := strings.TrimSpace(a.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1/messages"
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   req.Messages,
	}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		body["system"] = req.SystemPrompt
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("x-api-key", a.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, ErrProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, ErrProvider
	}

	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 128<<10)).Decode(&parsed); err != nil || len(parsed.Content) == 0 {
		return ChatResponse{}, ErrProvider
	}
	text := parsed.Content[0].Text
	tokens := parsed.Usage.InputTokens + parsed.Usage.OutputTokens
	if tokens <= 0 {
		tokens = estimateTokens(req.SystemPrompt, text)
	}
	return ChatResponse{
		Content:    text,
		TokensUsed: tokens,
		Provider:   "anthropic",
		Model:      model,
	}, nil
}

func estimateTokens(a, b string) int64 {
	chars := len(a) + len(b)
	if chars < 4 {
		return 10
	}
	return int64(chars / 4)
}
