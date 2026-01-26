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

type appointmentsAPIResponse struct {
	Embedded struct {
		Appointments []struct {
			AppointmentID   string  `json:"appointmentId"`
			Version         int64   `json:"version"`
			AppointmentDate string  `json:"appointmentDate"` // YYYY-MM-DD
			StartTime       string  `json:"startTime"`       // HH:MM:SS
			EndTime         string  `json:"endTime"`         // HH:MM:SS
			Price           float64 `json:"price"`

			StaffID   string `json:"staffId"`
			Confirmed bool   `json:"confirmed"`
			ServiceID string `json:"serviceId"`

			CreatedAt string `json:"createdAt"` // RFC3339
			UpdatedAt string `json:"updatedAt"` // RFC3339

			StaffRequest   bool   `json:"staffRequest"`
			PreferredStaff bool   `json:"preferredStaff"`
			ClientID       string `json:"clientId"`

			ServiceRewardID    string `json:"serviceRewardId"`
			PurchasingBranchID string `json:"purchasingBranchId"`
			ServiceName        string `json:"serviceName"`

			State           string `json:"state"`
			ActivationState string `json:"activationState"`

			DepositAmount   *float64 `json:"depositAmount"`
			DepositDateTime *string  `json:"depositDateTime"`

			BookingID string `json:"bookingId"`
			Source    string `json:"source"`

			Deleted bool `json:"deleted"`

			InternetServiceCategories []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"internetServiceCategories"`

			BranchID string `json:"branchId"`
		} `json:"appointments"`
	} `json:"_embedded"`

	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

type AppointmentsAPIClient struct {
	BaseURL  string
	User     string
	Pass     string
	Business string
	HTTP     *http.Client
}

func NewAppointmentsAPIClient(user, pass, business string) *AppointmentsAPIClient {
	return &AppointmentsAPIClient{
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

// FetchAppointmentsPage fetches a single page for one branch within a 1-month date range,
// optionally bounded by updated_from/updated_to.
func (c *AppointmentsAPIClient) FetchAppointmentsPage(
	ctx context.Context,
	branchID string,
	fromDate, toDate time.Time,
	updatedFrom *time.Time,
	page, size int,
	fetchCanceled, fetchDeleted, fetchArchived bool,
	fetchOnlineCategory bool,
) ([]models.AppointmentAPI, int, error) {

	if size <= 0 {
		size = 100
	}

	u, _ := url.Parse(fmt.Sprintf("%s/business/%s/branch/%s/appointment", c.BaseURL, c.Business, branchID))
	q := u.Query()

	q.Set("from_date", fromDate.Format("2006-01-02"))
	q.Set("to_date", toDate.Format("2006-01-02"))

	q.Set("size", fmt.Sprintf("%d", size))
	q.Set("page", fmt.Sprintf("%d", page))

	// include more complete history
	q.Set("fetch_canceled", fmt.Sprintf("%t", fetchCanceled))
	q.Set("fetch_deleted", fmt.Sprintf("%t", fetchDeleted))
	q.Set("fetch_archived", fmt.Sprintf("%t", fetchArchived))

	// categories are in your required field list
	q.Set("fetch_online_category", fmt.Sprintf("%t", fetchOnlineCategory))

	if updatedFrom != nil {
		// Phorest expects date-time strings; RFC3339 w/ millis is typically accepted
		q.Set("updated_from", updatedFrom.UTC().Format("2006-01-02T15:04:05.000Z"))
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
		return nil, 0, fmt.Errorf("appointments API status=%d, body=%s", resp.StatusCode, string(b))
	}

	var api appointmentsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, 0, err
	}

	parseDate := func(s string) (time.Time, error) {
		return time.Parse("2006-01-02", s)
	}
	parseTS := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
		} {
			if t, err := time.Parse(layout, s); err == nil {
				return &t
			}
		}
		return nil
	}

	out := make([]models.AppointmentAPI, 0, len(api.Embedded.Appointments))
	for _, a := range api.Embedded.Appointments {
		d, err := parseDate(a.AppointmentDate)
		if err != nil {
			// skip malformed rows rather than failing whole sync
			continue
		}

		var catsJSON string
		if fetchOnlineCategory {
			if b, err := json.Marshal(a.InternetServiceCategories); err == nil {
				catsJSON = string(b)
			} else {
				catsJSON = "[]"
			}
		} else {
			catsJSON = "[]"
		}

		out = append(out, models.AppointmentAPI{
			BranchID:      branchID,
			AppointmentID: a.AppointmentID,
			Version:       a.Version,

			AppointmentDate: d,
			StartTime:       a.StartTime,
			EndTime:         a.EndTime,

			Price:         a.Price,
			DepositAmount: a.DepositAmount,

			StaffID:   a.StaffID,
			Confirmed: a.Confirmed,
			ServiceID: a.ServiceID,

			CreatedAtPhorest: parseTS(a.CreatedAt),
			UpdatedAtPhorest: parseTS(a.UpdatedAt),

			StaffRequest:   a.StaffRequest,
			PreferredStaff: a.PreferredStaff,
			ClientID:       a.ClientID,

			ServiceRewardID:    a.ServiceRewardID,
			PurchasingBranchID: a.PurchasingBranchID,
			ServiceName:        a.ServiceName,

			State:           a.State,
			ActivationState: a.ActivationState,

			DepositDateTime: parseTS(derefString(a.DepositDateTime)),
			BookingID:       a.BookingID,
			Source:          a.Source,

			Deleted: a.Deleted,

			InternetServiceCategories: catsJSON,
		})
	}

	return out, api.Page.TotalPages, nil
}
