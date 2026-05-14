package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GoDaddy implements Provider using the GoDaddy API.
type GoDaddy struct {
	apiKey    string
	apiSecret string
	client    *http.Client
}

func NewGoDaddy(apiKey, apiSecret string) *GoDaddy {
	return &GoDaddy{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    &http.Client{Timeout: apiTimeout},
	}
}

const gdBase = "https://api.godaddy.com/v1"

type gdDomain struct {
	DomainID int    `json:"domainId"`
	Domain   string `json:"domain"`
}

type gdRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

func (g *GoDaddy) do(req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", g.apiKey, g.apiSecret))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("godaddy: %s - %s", errResp.Code, errResp.Message)
	}
	return nil
}

func (g *GoDaddy) doJSON(req *http.Request, v interface{}) error {
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", g.apiKey, g.apiSecret))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("godaddy: %s - %s", errResp.Code, errResp.Message)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (g *GoDaddy) ListDomains(ctx context.Context) ([]Domain, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", gdBase+"/domains", nil)
	if err != nil {
		return nil, err
	}
	var gds []gdDomain
	if err := g.doJSON(req, &gds); err != nil {
		return nil, err
	}
	domains := make([]Domain, len(gds))
	for i, d := range gds {
		domains[i] = Domain{ID: d.Domain, Name: d.Domain}
	}
	return domains, nil
}

func (g *GoDaddy) ListRecords(ctx context.Context, domainID string) ([]Record, error) {
	url := fmt.Sprintf("%s/domains/%s/records", gdBase, domainID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	var grs []gdRecord
	if err := g.doJSON(req, &grs); err != nil {
		return nil, err
	}
	records := make([]Record, len(grs))
	for i, r := range grs {
		records[i] = Record{
			ID:       fmt.Sprintf("%s/%s", r.Type, r.Name),
			Type:     r.Type,
			Name:     r.Name,
			Value:    r.Data,
			TTL:      r.TTL,
			Priority: r.Priority,
		}
	}
	return records, nil
}

func (g *GoDaddy) CreateRecord(ctx context.Context, domainID string, r Record) error {
	rec := []gdRecord{{
		Type:     r.Type,
		Name:     r.Name,
		Data:     r.Value,
		TTL:      r.TTL,
		Priority: r.Priority,
	}}
	data, _ := json.Marshal(rec)
	url := fmt.Sprintf("%s/domains/%s/records", gdBase, domainID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return g.do(req)
}

func (g *GoDaddy) UpdateRecord(ctx context.Context, domainID string, r Record) error {
	rec := []gdRecord{{
		Type:     r.Type,
		Name:     r.Name,
		Data:     r.Value,
		TTL:      r.TTL,
		Priority: r.Priority,
	}}
	data, _ := json.Marshal(rec)
	url := fmt.Sprintf("%s/domains/%s/records/%s/%s", gdBase, domainID, r.Type, r.Name)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return g.do(req)
}

func (g *GoDaddy) DeleteRecord(ctx context.Context, domainID string, recordID string) error {
	parts := strings.SplitN(recordID, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("godaddy: invalid record ID %q (expected type/name)", recordID)
	}
	url := fmt.Sprintf("%s/domains/%s/records/%s/%s", gdBase, domainID, parts[0], parts[1])
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	return g.do(req)
}
