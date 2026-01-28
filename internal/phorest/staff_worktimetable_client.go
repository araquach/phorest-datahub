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

type staffWorkTimetableResponse struct {
	Embedded struct {
		WorkTimeTables []struct {
			StaffID   string `json:"staffId"`
			BranchID  string `json:"branchId"`
			TimeSlots []struct {
				Date           string  `json:"date"`      // YYYY-MM-DD
				StartTime      string  `json:"startTime"` // HH:MM:SS
				EndTime        string  `json:"endTime"`   // HH:MM:SS
				TimeOffStart   *string `json:"timeOffStartTime"`
				TimeOffEnd     *string `json:"timeOffEndTime"`
				Type           string  `json:"type"`
				Custom         *string `json:"custom"`
				BranchID       *string `json:"branchId"`
				WorkActivityID *string `json:"workActivityId"`
			} `json:"timeSlots"`
		} `json:"workTimeTables"`
	} `json:"_embedded"`

	Page struct {
		TotalPages int `json:"totalPages"`
		Number     int `json:"number"`
		Size       int `json:"size"`
	} `json:"page"`
}

type StaffWorkTimetableClient struct {
	BaseURL  string
	User     string
	Pass     string
	Business string
	HTTP     *http.Client
}

func NewStaffWorkTimetableClient(user, pass, business string) *StaffWorkTimetableClient {
	return &StaffWorkTimetableClient{
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

func (c *StaffWorkTimetableClient) FetchWorkTimetablePage(
	ctx context.Context,
	branchID string,
	fromDate, toDate time.Time,
	activityType *string,
	page, size int,
) ([]models.StaffWorkTimetableSlot, int, error) {

	if size <= 0 {
		size = 100
	}
	if size > 100 {
		size = 100
	}

	u, _ := url.Parse(fmt.Sprintf("%s/business/%s/branch/%s/staff/worktimetable", c.BaseURL, c.Business, branchID))
	q := u.Query()
	q.Set("from_date", fromDate.Format("2006-01-02"))
	q.Set("to_date", toDate.Format("2006-01-02"))
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("size", fmt.Sprintf("%d", size))
	if activityType != nil && *activityType != "" {
		q.Set("activity_type", *activityType)
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
		return nil, 0, fmt.Errorf("worktimetable API status=%d, body=%s", resp.StatusCode, string(b))
	}

	var api staffWorkTimetableResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, 0, err
	}

	out := make([]models.StaffWorkTimetableSlot, 0, 5000)

	for _, tt := range api.Embedded.WorkTimeTables {
		for _, s := range tt.TimeSlots {
			d, err := time.Parse("2006-01-02", s.Date)
			if err != nil {
				continue
			}
			
			out = append(out, models.StaffWorkTimetableSlot{
				BranchID: branchID,
				StaffID:  tt.StaffID,

				SlotDate:  dateOnly(d.UTC()),
				StartTime: s.StartTime,
				EndTime:   s.EndTime,

				TimeOffStartTime: s.TimeOffStart,
				TimeOffEndTime:   s.TimeOffEnd,

				Type:           s.Type,
				Custom:         s.Custom,
				SlotBranchID:   s.BranchID,
				WorkActivityID: s.WorkActivityID,
			})
		}
	}

	return out, api.Page.TotalPages, nil
}
