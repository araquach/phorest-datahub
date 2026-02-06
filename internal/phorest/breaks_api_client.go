package phorest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
)

type breaksAPIResponse struct {
	Embedded struct {
		Breaks []struct {
			BreakDate string `json:"breakDate"` // YYYY-MM-DD
			StartTime string `json:"startTime"` // HH:MM:SS
			EndTime   string `json:"endTime"`   // HH:MM:SS

			StaffID   string  `json:"staffId"`
			RoomID    *string `json:"roomId"`
			MachineID *string `json:"machineId"`
			Label     *string `json:"label"`
			PaidBreak bool    `json:"paidBreak"`

			BreakID string `json:"breakId"`
			Version int64  `json:"version"`
		} `json:"breaks"`
	} `json:"_embedded"`

	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

type BreaksAPIClient struct {
	BaseURL  string
	User     string
	Pass     string
	Business string
	HTTP     *http.Client
}

func NewBreaksAPIClient(user, pass, business string) *BreaksAPIClient {
	return &BreaksAPIClient{
		BaseURL:  "https://api-gateway-eu.phorest.com/third-party-api-server/api",
		User:     user,
		Pass:     pass,
		Business: business,
		HTTP: &http.Client{
			Timeout: 25 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// FetchBreaksPage fetches a single page for one branch within a 1-month date range.
// Optional filters (staff/room/machine) are supported, but you can pass nil for all.
func (c *BreaksAPIClient) FetchBreaksPage(
	ctx context.Context,
	branchID string,
	fromDate, toDate time.Time,
	page, size int,
	staffID, roomID, machineID *string,
) ([]models.BreakAPI, int, error) {

	if size <= 0 {
		size = 100
	}
	if size > 100 {
		size = 100
	}

	u, _ := url.Parse(fmt.Sprintf("%s/business/%s/branch/%s/break", c.BaseURL, c.Business, branchID))
	q := u.Query()

	q.Set("from_date", fromDate.Format("2006-01-02"))
	q.Set("to_date", toDate.Format("2006-01-02"))
	q.Set("size", fmt.Sprintf("%d", size))
	q.Set("page", fmt.Sprintf("%d", page))

	if staffID != nil && *staffID != "" {
		q.Set("staff_id", *staffID)
	}
	if roomID != nil && *roomID != "" {
		q.Set("room_id", *roomID)
	}
	if machineID != nil && *machineID != "" {
		q.Set("machine_id", *machineID)
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(c.User, c.Pass)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("breaks API status=%d, body=%s", resp.StatusCode, string(b))
	}

	var api breaksAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, 0, err
	}

	parseDate := func(s string) (time.Time, error) {
		// store as date-only UTC
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}

	out := make([]models.BreakAPI, 0, len(api.Embedded.Breaks))
	for _, b := range api.Embedded.Breaks {
		d, err := parseDate(b.BreakDate)
		if err != nil {
			continue
		}
		out = append(out, models.BreakAPI{
			BranchID:  branchID,
			BreakID:   b.BreakID,
			Version:   b.Version,
			BreakDate: d,
			StartTime: b.StartTime,
			EndTime:   b.EndTime,
			StaffID:   b.StaffID,
			RoomID:    b.RoomID,
			MachineID: b.MachineID,
			Label:     b.Label,
			PaidBreak: b.PaidBreak,
		})
	}

	return out, api.Page.TotalPages, nil
}
