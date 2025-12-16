package phorest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
)

type clientsAPIResponse struct {
	Embedded struct {
		Clients []struct {
			LinkedClientMobile string `json:"linkedClientMobile"`
			LandLine           string `json:"landLine"`
			Email              string `json:"email"`
			Address            struct {
				StreetAddress1 string `json:"streetAddress1"`
				StreetAddress2 string `json:"streetAddress2"`
				City           string `json:"city"`
				State          string `json:"state"`
				PostalCode     string `json:"postalCode"`
				Country        string `json:"country"`
			} `json:"address"`
			BirthDate       *string `json:"birthDate"`
			ClientSince     *string `json:"clientSince"`
			Gender          string  `json:"gender"`
			Notes           string  `json:"notes"`
			SMSMkConsent    bool    `json:"smsMarketingConsent"`
			EmailMkConsent  bool    `json:"emailMarketingConsent"`
			SMSRemConsent   bool    `json:"smsReminderConsent"`
			EmailRemConsent bool    `json:"emailReminderConsent"`

			PreferredStaffID  string   `json:"preferredStaffId"`
			ExternalID        string   `json:"externalId"`
			CreatingBranchID  string   `json:"creatingBranchId"`
			Archived          bool     `json:"archived"`
			Banned            bool     `json:"banned"`
			ClientCategoryIDs []string `json:"clientCategoryIds"`

			ClientID  string `json:"clientId"`
			Version   int64  `json:"version"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Mobile    string `json:"mobile"`

			CreatedAt *string `json:"createdAt"`
			UpdatedAt *string `json:"updatedAt"`

			PhotoURL *string `json:"photoUrl"`

			CreditAccount *struct {
				OutstandingBalance *float64 `json:"outstandingBalance"`
				CreditDays         *int64   `json:"creditDays"`
				CreditLimit        *float64 `json:"creditLimit"`
			} `json:"creditAccount"`

			LoyaltyCard *struct {
				Serial string  `json:"serial"`
				Points float64 `json:"points"`
			} `json:"loyaltyCard"`

			MergedToClientID string  `json:"mergedToClientId"`
			Deleted          bool    `json:"deleted"`
			FirstVisit       *string `json:"firstVisit"`
			LastVisit        *string `json:"lastVisit"`
		} `json:"clients"`
	} `json:"_embedded"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

type ClientsAPIClient struct {
	BaseURL  string
	User     string
	Pass     string
	Business string
	HTTP     *http.Client
}

func NewClientsAPIClient(user, pass, business string) *ClientsAPIClient {
	return &ClientsAPIClient{
		BaseURL:  "https://api-gateway-eu.phorest.com/third-party-api-server/api",
		User:     user,
		Pass:     pass,
		Business: business,
		HTTP: &http.Client{
			Timeout: 20 * time.Second,
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

func (c *ClientsAPIClient) FetchClientsPage(
	ctx context.Context,
	updatedAfter *time.Time,
	page, size int,
) ([]models.ClientAPI, int, error) {

	if size <= 0 {
		size = 100
	}

	u, _ := url.Parse(fmt.Sprintf("%s/business/%s/client", c.BaseURL, c.Business))
	q := u.Query()
	q.Set("size", fmt.Sprintf("%d", size))
	q.Set("page", fmt.Sprintf("%d", page))
	if updatedAfter != nil {
		q.Set("updatedAfter", updatedAfter.UTC().Format("2006-01-02T15:04:05.000Z"))
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
		return nil, 0, fmt.Errorf("clients API status=%d, body=%s", resp.StatusCode, string(b))
	}

	var api clientsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, 0, err
	}

	parseTime := func(s *string) *time.Time {
		if s == nil || *s == "" {
			return nil
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(layout, *s); err == nil {
				return &t
			}
		}
		return nil
	}

	out := make([]models.ClientAPI, 0, len(api.Embedded.Clients))
	for _, c0 := range api.Embedded.Clients {
		var lp *float64
		if c0.LoyaltyCard != nil {
			lp = &c0.LoyaltyCard.Points
		}
		var serial string
		if c0.LoyaltyCard != nil {
			serial = c0.LoyaltyCard.Serial
		}

		var bal, limit *float64
		var days *int64
		if c0.CreditAccount != nil {
			bal = c0.CreditAccount.OutstandingBalance
			limit = c0.CreditAccount.CreditLimit
			days = c0.CreditAccount.CreditDays
		}

		catIDs := ""
		if len(c0.ClientCategoryIDs) > 0 {
			catIDs = strings.Join(c0.ClientCategoryIDs, ",")
		}

		out = append(out, models.ClientAPI{
			ClientID:                 c0.ClientID,
			Version:                  c0.Version,
			FirstName:                c0.FirstName,
			LastName:                 c0.LastName,
			Mobile:                   c0.Mobile,
			LinkedClientMobile:       c0.LinkedClientMobile,
			LandLine:                 c0.LandLine,
			Email:                    c0.Email,
			StreetAddress1:           c0.Address.StreetAddress1,
			StreetAddress2:           c0.Address.StreetAddress2,
			City:                     c0.Address.City,
			State:                    c0.Address.State,
			PostalCode:               c0.Address.PostalCode,
			Country:                  c0.Address.Country,
			BirthDate:                parseTime(c0.BirthDate),
			ClientSince:              parseTime(c0.ClientSince),
			Gender:                   c0.Gender,
			Notes:                    c0.Notes,
			SMSMarketingConsent:      c0.SMSMkConsent,
			EmailMarketingConsent:    c0.EmailMkConsent,
			SMSReminderConsent:       c0.SMSRemConsent,
			EmailReminderConsent:     c0.EmailRemConsent,
			PreferredStaffID:         c0.PreferredStaffID,
			ExternalID:               c0.ExternalID,
			CreatingBranchID:         c0.CreatingBranchID,
			Archived:                 c0.Archived,
			Banned:                   c0.Banned,
			ClientCategoryIDs:        catIDs,
			CreatedAtPhorest:         parseTime(c0.CreatedAt),
			UpdatedAtPhorest:         parseTime(c0.UpdatedAt),
			PhotoURL:                 derefString(c0.PhotoURL),
			LoyaltyCardSerial:        serial,
			LoyaltyPoints:            lp,
			CreditOutstandingBalance: bal,
			CreditLimit:              limit,
			CreditDays:               days,
			MergedToClientID:         c0.MergedToClientID,
			Deleted:                  c0.Deleted,
			FirstVisit:               parseTime(c0.FirstVisit),
			LastVisit:                parseTime(c0.LastVisit),
		})
	}

	return out, api.Page.TotalPages, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
