package midtrans

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var _ Client = (*HTTPClient)(nil)

type HTTPClient struct {
	serverKey   string
	snapBaseURL string
	apiBaseURL  string
	httpClient  *http.Client
}

func NewHTTPClient(serverKey string, isProduction bool, baseURLOverride string) *HTTPClient {
	snapBase, apiBase := "https://app.sandbox.midtrans.com", "https://api.sandbox.midtrans.com"
	if isProduction {
		snapBase, apiBase = "https://app.midtrans.com", "https://api.midtrans.com"
	}
	if baseURLOverride != "" {
		snapBase, apiBase = baseURLOverride, baseURLOverride
	}
	return &HTTPClient{
		serverKey:   serverKey,
		snapBaseURL: snapBase,
		apiBaseURL:  apiBase,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *HTTPClient) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.serverKey+":"))
}

func (c *HTTPClient) do(ctx context.Context, method, url string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("midtrans request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &StatusCodeError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *HTTPClient) CreateTransaction(ctx context.Context, req TransactionRequest) (TransactionResponse, error) {
	var out TransactionResponse
	err := c.do(ctx, http.MethodPost, c.snapBaseURL+"/snap/v1/transactions", req, &out)
	return out, err
}

func (c *HTTPClient) GetStatus(ctx context.Context, orderID string) (StatusResponse, error) {
	var out StatusResponse
	err := c.do(ctx, http.MethodGet, c.apiBaseURL+"/v2/"+orderID+"/status", nil, &out)
	return out, err
}

func (c *HTTPClient) ExpireTransaction(ctx context.Context, orderID string) error {
	return c.do(ctx, http.MethodPost, c.apiBaseURL+"/v2/"+orderID+"/expire", nil, nil)
}
