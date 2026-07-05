package radiusctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP API client used by every radiusctl subcommand.
// It owns no state beyond its config: every call hits the server, so the
// binary is safe to run from any host that can reach the API.
type Client struct {
	server string // base URL, e.g. "http://localhost:8083"
	secret string // bearer token
	http   *http.Client
}

// NewClient returns a configured API client.
func NewClient(server, secret string) *Client {
	return &Client{
		server: server,
		secret: secret,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// apiError is returned for any non-success envelope or non-2xx status.
type apiError struct {
	HTTP    int
	Code    string
	Message string
}

func (e *apiError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("api error %d %s: %s", e.HTTP, e.Code, e.Message)
	}
	return fmt.Sprintf("api error %d: %s", e.HTTP, e.Message)
}

// do performs an HTTP round-trip and unwraps the response envelope. On
// success it unmarshals the response Data field into out (when out is
// non-nil). On failure it returns an *apiError carrying the server code
// and message.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.server+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// Not JSON: fall back to the raw payload in the error message.
		return &apiError{HTTP: resp.StatusCode, Message: fmt.Sprintf("non-JSON response: %s", truncated(raw, 200))}
	}

	if !env.Success {
		msg := "request failed"
		code := ""
		if env.Err != nil {
			msg = env.Err.Message
			code = env.Err.Code
		}
		return &apiError{HTTP: resp.StatusCode, Code: code, Message: msg}
	}

	if out != nil {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("unmarshal response data: %w", err)
		}
	}
	return nil
}

func truncated(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// nopReader returns nil for body-less requests. Inline helper keeps do() clean.
func nopReader() io.Reader { return nil }

// --- Status ---

func (c *Client) GetStatus(ctx context.Context) (Status, error) {
	var s Status
	err := c.do(ctx, http.MethodGet, "/api/v1/radius/status", nil, &s)
	return s, err
}

// --- NAS ---

func (c *Client) ListNAS(ctx context.Context) ([]NAS, error) {
	var out []NAS
	err := c.do(ctx, http.MethodGet, "/api/v1/radius/nases", nil, &out)
	return out, err
}

func (c *Client) CreateNAS(ctx context.Context, req CreateNASRequest) (NAS, error) {
	var out NAS
	err := c.do(ctx, http.MethodPost, "/api/v1/radius/nases", req, &out)
	return out, err
}

func (c *Client) UpdateNAS(ctx context.Context, id string, req UpdateNASRequest) (NAS, error) {
	var out NAS
	err := c.do(ctx, http.MethodPut, "/api/v1/radius/nases/"+id, req, &out)
	return out, err
}

func (c *Client) DeleteNAS(ctx context.Context, id string) (DeleteResult, error) {
	var out DeleteResult
	err := c.do(ctx, http.MethodDelete, "/api/v1/radius/nases/"+id, nil, &out)
	return out, err
}

// --- Subscriber ---

func (c *Client) ListSubscribers(ctx context.Context) ([]Subscriber, error) {
	var out []Subscriber
	err := c.do(ctx, http.MethodGet, "/api/v1/radius/subscribers", nil, &out)
	return out, err
}

func (c *Client) CreateSubscriber(ctx context.Context, req CreateSubscriberRequest) (Subscriber, error) {
	var out Subscriber
	err := c.do(ctx, http.MethodPost, "/api/v1/radius/subscribers", req, &out)
	return out, err
}

func (c *Client) UpdateSubscriber(ctx context.Context, id string, req UpdateSubscriberRequest) (Subscriber, error) {
	var out Subscriber
	err := c.do(ctx, http.MethodPut, "/api/v1/radius/subscribers/"+id, req, &out)
	return out, err
}

func (c *Client) DeleteSubscriber(ctx context.Context, id string) (DeleteResult, error) {
	var out DeleteResult
	err := c.do(ctx, http.MethodDelete, "/api/v1/radius/subscribers/"+id, nil, &out)
	return out, err
}

// --- Session ---

func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	var out []Session
	err := c.do(ctx, http.MethodGet, "/api/v1/radius/sessions", nil, &out)
	return out, err
}

func (c *Client) DisconnectUser(ctx context.Context, req DisconnectRequest) (map[string]any, error) {
	var out map[string]any
	err := c.do(ctx, http.MethodPost, "/api/v1/radius/sessions/disconnect", req, &out)
	return out, err
}

func (c *Client) CoAChange(ctx context.Context, req CoAChangeRequest) (CoaChangeResult, error) {
	var out CoaChangeResult
	err := c.do(ctx, http.MethodPost, "/api/v1/radius/subscribers/coa-change", req, &out)
	return out, err
}

func (c *Client) CleanupSessions(ctx context.Context) (CleanupResult, error) {
	var out CleanupResult
	err := c.do(ctx, http.MethodPost, "/api/v1/radius/sessions/cleanup", nil, &out)
	return out, err
}

func (c *Client) ReconcileSessions(ctx context.Context) (map[string]int, error) {
	var out map[string]int
	err := c.do(ctx, http.MethodPost, "/api/v1/radius/sessions/reconcile", nil, &out)
	return out, err
}

// --- Voucher packages ---

func (c *Client) ListVoucherPackages(ctx context.Context) ([]VoucherPackage, error) {
	var out []VoucherPackage
	err := c.do(ctx, http.MethodGet, "/api/v1/voucher-packages", nil, &out)
	return out, err
}

func (c *Client) CreateVoucherPackage(ctx context.Context, req CreateVoucherPackageRequest) (VoucherPackage, error) {
	var out VoucherPackage
	err := c.do(ctx, http.MethodPost, "/api/v1/voucher-packages", req, &out)
	return out, err
}

func (c *Client) UpdateVoucherPackage(ctx context.Context, id string, req UpdateVoucherPackageRequest) (VoucherPackage, error) {
	var out VoucherPackage
	err := c.do(ctx, http.MethodPut, "/api/v1/voucher-packages/"+id, req, &out)
	return out, err
}

func (c *Client) DeleteVoucherPackage(ctx context.Context, id string) (DeleteResult, error) {
	var out DeleteResult
	err := c.do(ctx, http.MethodDelete, "/api/v1/voucher-packages/"+id, nil, &out)
	return out, err
}

// --- Vouchers ---

func (c *Client) ListVouchers(ctx context.Context) ([]Subscriber, error) {
	var out []Subscriber
	err := c.do(ctx, http.MethodGet, "/api/v1/vouchers", nil, &out)
	return out, err
}

func (c *Client) GenerateVouchers(ctx context.Context, req GenerateVoucherRequest) ([]GeneratedVoucher, error) {
	var out []GeneratedVoucher
	err := c.do(ctx, http.MethodPost, "/api/v1/vouchers/generate", req, &out)
	return out, err
}

func (c *Client) VoucherBalance(ctx context.Context, code string) (VoucherBalance, error) {
	var out VoucherBalance
	err := c.do(ctx, http.MethodGet, "/api/v1/vouchers/"+code+"/balance", nil, &out)
	return out, err
}
