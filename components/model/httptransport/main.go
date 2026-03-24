package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

type SensitiveFields struct {
	APIKey    string
	AuthToken string
}

func (s SensitiveFields) Mask() SensitiveFields {
	return SensitiveFields{
		APIKey:    maskString(s.APIKey),
		AuthToken: maskString(s.AuthToken),
	}
}

func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

type HTTPTransportLogger struct {
	transport     http.RoundTripper
	sensitive     SensitiveFields
	logRequest    bool
	logResponse   bool
	maskSensitive bool
}

type HTTPTransportLoggerConfig struct {
	Sensitive     SensitiveFields
	LogRequest    bool
	LogResponse   bool
	MaskSensitive bool
}

func NewHTTPTransportLogger(config *HTTPTransportLoggerConfig) *HTTPTransportLogger {
	return &HTTPTransportLogger{
		transport:     http.DefaultTransport,
		sensitive:     config.Sensitive,
		logRequest:    config.LogRequest,
		logResponse:   config.LogResponse,
		maskSensitive: config.MaskSensitive,
	}
}

func (t *HTTPTransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.logRequest {
		t.logHTTPRequest(req)
	}

	start := time.Now()
	resp, err := t.transport.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("[HTTP Error] %s %s - Error: %v (%.2fs)\n", req.Method, req.URL, err, duration.Seconds())
		return nil, err
	}

	if t.logResponse {
		t.logHTTPResponse(resp, duration)
	}

	return resp, nil
}

func (t *HTTPTransportLogger) logHTTPRequest(req *http.Request) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("[HTTP Request] %s %s\n", req.Method, req.URL)
	fmt.Println(strings.Repeat("-", 60))

	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		fmt.Printf("Failed to dump request: %v\n", err)
		return
	}

	dumpStr := string(dump)

	if t.maskSensitive {
		if t.sensitive.APIKey != "" {
			dumpStr = strings.ReplaceAll(dumpStr, t.sensitive.APIKey, t.sensitive.Mask().APIKey)
		}
		if t.sensitive.AuthToken != "" {
			dumpStr = strings.ReplaceAll(dumpStr, t.sensitive.AuthToken, t.sensitive.Mask().AuthToken)
		}
	}

	fmt.Println(dumpStr)
}

func (t *HTTPTransportLogger) logHTTPResponse(resp *http.Response, duration time.Duration) {
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("[HTTP Response] Status: %s (%.2fs)\n", resp.Status, duration.Seconds())

	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		fmt.Printf("Failed to dump response headers: %v\n", err)
		return
	}

	fmt.Println(string(dump))

	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Failed to read response body: %v\n", err)
			return
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
		} else {
			fmt.Println(string(body))
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

type LoggedChatModel struct {
	*openai.ChatModel
	logger *HTTPTransportLogger
}

func NewLoggedChatModel(ctx context.Context, config *openai.ChatModelConfig, loggerConfig *HTTPTransportLoggerConfig) (*LoggedChatModel, error) {
	apiKey := config.APIKey

	logger := NewHTTPTransportLogger(loggerConfig)
	logger.sensitive.APIKey = apiKey

	httpClient := &http.Client{
		Transport: logger,
		Timeout:   60 * time.Second,
	}

	modelConfig := *config
	modelConfig.HTTPClient = httpClient

	model, err := openai.NewChatModel(ctx, &modelConfig)
	if err != nil {
		return nil, err
	}

	return &LoggedChatModel{
		ChatModel: model,
		logger:    logger,
	}, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	ctx := context.Background()

	baseURL := getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("OPENAI_API_KEY is required")
	}

	fmt.Println("=== HTTP Transport Logger Demo ===")
	fmt.Println("This demo logs all HTTP requests/responses in cURL-like format")
	fmt.Println()

	loggerConfig := &HTTPTransportLoggerConfig{
		Sensitive: SensitiveFields{
			APIKey: apiKey,
		},
		LogRequest:    true,
		LogResponse:   true,
		MaskSensitive: true,
	}

	model, err := NewLoggedChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
	}, loggerConfig)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Test 1: Non-streaming request ===")
	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("Say 'Hello, World!' and nothing else."),
	}

	msg, err := model.Generate(ctx, messages)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("\nFinal response: %s\n", msg.Content)
	}

	fmt.Println("\n\n=== Test 2: Streaming request ===")
	sr, err := model.Stream(ctx, messages)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer sr.Close()

	fmt.Println("\nStream response:")
	for {
		chunk, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			break
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println("\n\n=== Demo Complete ===")
	fmt.Println("Note: API keys are masked for security in the logs above")
}
