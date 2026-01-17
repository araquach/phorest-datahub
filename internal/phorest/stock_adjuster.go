package phorest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/services"
)

type StockAdjuster struct {
	BaseURL    string // e.g. https://api-gateway-eu.phorest.com/third-party-api-server
	BusinessID string
	Username   string
	Password   string
	HTTP       *http.Client
}

func NewStockAdjuster(baseURL, businessID, username, password string) StockAdjuster {
	baseURL = strings.TrimRight(baseURL, "/")
	return StockAdjuster{
		BaseURL:    baseURL,
		BusinessID: businessID,
		Username:   username,
		Password:   password,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a StockAdjuster) AdjustStock(ctx context.Context, branchID string, req services.StockAdjustmentRequest) error {
	if branchID == "" {
		return fmt.Errorf("branchID is required")
	}
	if a.BusinessID == "" {
		return fmt.Errorf("businessID is required")
	}
	if a.Username == "" || a.Password == "" {
		return fmt.Errorf("phorest username/password required")
	}
	if a.HTTP == nil {
		return fmt.Errorf("http client is nil")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/business/%s/branch/%s/stock/adjustment", a.BaseURL, a.BusinessID, branchID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Basic auth (Phorest third-party API typically uses Basic)
	b64 := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
	httpReq.Header.Set("Authorization", "Basic "+b64)

	resp, err := a.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("phorest adjust stock failed: status=%s body=%s", resp.Status, string(b))
	}

	return nil
}
