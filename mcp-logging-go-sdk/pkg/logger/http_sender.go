package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPSender struct {
	client         *http.Client
	serverURL      string
	headers        map[string]string
	retryer        *retryer
	circuitBreaker *CircuitBreaker
}

func NewHTTPSender(serverURL string, timeout time.Duration) *HTTPSender {
	retryConfig := RetryConfig{
		InitialInterval:     1 * time.Second,
		MaxInterval:         30 * time.Second,
		MaxElapsedTime:      5 * time.Minute,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}
	return &HTTPSender{
		client: &http.Client{
			Timeout: timeout,
		},
		serverURL: serverURL + "/api/logs",
		headers: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "mcp-logging-go-sdk/1.0.0",
		},
		retryer:        newRetryer(retryConfig),
		circuitBreaker: NewCircuitBreaker(5, 60*time.Second),
	}
}

func (h *HTTPSender) Send(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	payload := struct {
		Logs []LogEntry `json:"logs"`
	}{
		Logs: entries,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ErrServerError("failed to marshal log entries", err)
	}

	return h.circuitBreaker.Do(ctx, func() error {
		return h.retryer.Do(ctx, func() error {
			req, err := http.NewRequestWithContext(ctx, "POST", h.serverURL, bytes.NewReader(data))
			if err != nil {
				return ErrNetworkError("failed to create request", err)
			}

			for key, value := range h.headers {
				req.Header.Set(key, value)
			}

			resp, err := h.client.Do(req)
			if err != nil {
				return ErrNetworkError("failed to send request", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				body, _ := io.ReadAll(resp.Body)
				return ErrServerError(
					fmt.Sprintf("server returned status %d", resp.StatusCode),
					fmt.Errorf("response body: %s", string(body)),
				)
			}

			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				return &Error{
					Type:    ErrTypeServerError,
					Message: fmt.Sprintf("client error: status %d", resp.StatusCode),
					Err:     fmt.Errorf("response body: %s", string(body)),
				}
			}

			return nil
		})
	})
}

func (h *HTTPSender) HealthCheck(ctx context.Context) error {
	healthURL := h.serverURL + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return ErrNetworkError("failed to create health check request", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return ErrNetworkError("health check failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrServerError(
			fmt.Sprintf("health check failed with status %d", resp.StatusCode),
			nil,
		)
	}

	return nil
}

func (h *HTTPSender) Close() error {
	return nil
}
