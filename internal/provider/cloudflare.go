package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Cloudflare implements Provider using the Cloudflare API.
type Cloudflare struct {
	token  string
	client *http.Client
}

func NewCloudflare(token string) *Cloudflare {
	return &Cloudflare{
		token:  token,
		client: &http.Client{Timeout: apiTimeout},
	}
}

const cfBase = "https://api.cloudflare.com/client/v4"

type cfResponse struct {
	Success bool            `json:"success"`
	Errors  []cfError       `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cfRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
}

func (c *Cloudflare) do(req *http.Request) (*cfResponse, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	var r cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if !r.Success {
		msgs := make([]string, len(r.Errors))
		for i, e := range r.Errors {
			msgs[i] = e.Message
		}
		return nil, fmt.Errorf("cloudflare: %s", strings.Join(msgs, "; "))
	}
	return &r, nil
}

func (c *Cloudflare) ListDomains(ctx context.Context) ([]Domain, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", cfBase+"/zones?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	r, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var zones []cfZone
	if err := json.Unmarshal(r.Result, &zones); err != nil {
		return nil, err
	}
	domains := make([]Domain, len(zones))
	for i, z := range zones {
		domains[i] = Domain(z)
	}
	return domains, nil
}

func (c *Cloudflare) ListRecords(ctx context.Context, domainID string) ([]Record, error) {
	var allRecords []Record
	page := 1
	for {
		url := fmt.Sprintf("%s/zones/%s/dns_records?per_page=100&page=%d", cfBase, domainID, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		r, err := c.do(req)
		if err != nil {
			return nil, err
		}
		var recs []cfRecord
		if err := json.Unmarshal(r.Result, &recs); err != nil {
			return nil, err
		}
		if len(recs) == 0 {
			break
		}
		for _, rec := range recs {
			pri := 0
			if rec.Priority != nil {
				pri = *rec.Priority
			}
			allRecords = append(allRecords, Record{
				ID:       rec.ID,
				Type:     rec.Type,
				Name:     rec.Name,
				Value:    rec.Content,
				TTL:      rec.TTL,
				Priority: pri,
			})
		}
		if len(recs) < 100 {
			break
		}
		page++
	}
	return allRecords, nil
}

func (c *Cloudflare) CreateRecord(ctx context.Context, domainID string, r Record) error {
	body := cfRecord{
		Type:    r.Type,
		Name:    r.Name,
		Content: r.Value,
		TTL:     r.TTL,
	}
	if r.Type == "MX" || r.Type == "SRV" {
		body.Priority = &r.Priority
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", cfBase+"/zones/"+domainID+"/dns_records", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

func (c *Cloudflare) UpdateRecord(ctx context.Context, domainID string, r Record) error {
	body := cfRecord{
		Type:    r.Type,
		Name:    r.Name,
		Content: r.Value,
		TTL:     r.TTL,
	}
	if r.Type == "MX" || r.Type == "SRV" {
		body.Priority = &r.Priority
	}
	data, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cfBase, domainID, r.ID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

func (c *Cloudflare) DeleteRecord(ctx context.Context, domainID string, recordID string) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cfBase, domainID, recordID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}
