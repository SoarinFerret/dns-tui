package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const apiTimeout = 10 * time.Second

// DNSMadeEasy implements Provider using the DNS Made Easy API.
type DNSMadeEasy struct {
	apiKey    string
	apiSecret string
	client    *http.Client
}

func NewDNSMadeEasy(apiKey, apiSecret string) *DNSMadeEasy {
	return &DNSMadeEasy{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    &http.Client{Timeout: apiTimeout},
	}
}

const dmeBase = "https://api.dnsmadeeasy.com/V2.0"

type dmeDomain struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type dmeDomainsResponse struct {
	Data []dmeDomain `json:"data"`
}

type dmeRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	MXLevel  int    `json:"mxLevel"`
	Priority int    `json:"priority"`
}

type dmeRecordsResponse struct {
	Data []dmeRecord `json:"data"`
}

// httpDateFormat produces RFC 2616 HTTP-date with literal "GMT" suffix,
// e.g. "Sat, 12 Feb 2011 20:59:04 GMT".
const httpDateFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func (d *DNSMadeEasy) sign(req *http.Request) {
	now := time.Now().UTC().Format(httpDateFormat)
	mac := hmac.New(sha1.New, []byte(d.apiSecret))
	mac.Write([]byte(now))
	sig := hex.EncodeToString(mac.Sum(nil))

	// Set headers via direct map access to preserve exact casing.
	// Go's Header.Set() canonicalizes names (e.g. "x-dnsme-apiKey"
	// becomes "X-Dnsme-Apikey"), which DME's server rejects.
	req.Header["x-dnsme-apiKey"] = []string{d.apiKey}
	req.Header["x-dnsme-requestDate"] = []string{now}
	req.Header["x-dnsme-hmac"] = []string{sig}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

func (d *DNSMadeEasy) doJSON(req *http.Request, v interface{}) error {
	d.sign(req)
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		var errBody struct {
			Error []string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("dnsmadeeasy: %s", strings.Join(errBody.Error, "; "))
	}
	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

func (d *DNSMadeEasy) ListDomains(ctx context.Context) ([]Domain, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", dmeBase+"/dns/managed", nil)
	if err != nil {
		return nil, err
	}
	var resp dmeDomainsResponse
	if err := d.doJSON(req, &resp); err != nil {
		return nil, err
	}
	domains := make([]Domain, len(resp.Data))
	for i, dm := range resp.Data {
		domains[i] = Domain{ID: fmt.Sprintf("%d", dm.ID), Name: dm.Name}
	}
	return domains, nil
}

func (d *DNSMadeEasy) ListRecords(ctx context.Context, domainID string) ([]Record, error) {
	url := fmt.Sprintf("%s/dns/managed/%s/records", dmeBase, domainID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	var resp dmeRecordsResponse
	if err := d.doJSON(req, &resp); err != nil {
		return nil, err
	}
	records := make([]Record, len(resp.Data))
	for i, r := range resp.Data {
		pri := r.MXLevel
		if r.Priority != 0 {
			pri = r.Priority
		}
		records[i] = Record{
			ID:       fmt.Sprintf("%d", r.ID),
			Type:     r.Type,
			Name:     r.Name,
			Value:    r.Value,
			TTL:      r.TTL,
			Priority: pri,
		}
	}
	return records, nil
}

func (d *DNSMadeEasy) CreateRecord(ctx context.Context, domainID string, r Record) error {
	body := map[string]interface{}{
		"type":  r.Type,
		"name":  r.Name,
		"value": r.Value,
		"ttl":   r.TTL,
	}
	switch r.Type {
	case "MX":
		body["mxLevel"] = r.Priority
	case "SRV":
		body["priority"] = r.Priority
	}
	data, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/dns/managed/%s/records", dmeBase, domainID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return d.doJSON(req, nil)
}

func (d *DNSMadeEasy) UpdateRecord(ctx context.Context, domainID string, r Record) error {
	body := map[string]interface{}{
		"id":    r.ID,
		"type":  r.Type,
		"name":  r.Name,
		"value": r.Value,
		"ttl":   r.TTL,
	}
	switch r.Type {
	case "MX":
		body["mxLevel"] = r.Priority
	case "SRV":
		body["priority"] = r.Priority
	}
	data, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/dns/managed/%s/records/%s", dmeBase, domainID, r.ID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return d.doJSON(req, nil)
}

func (d *DNSMadeEasy) DeleteRecord(ctx context.Context, domainID string, recordID string) error {
	url := fmt.Sprintf("%s/dns/managed/%s/records/%s", dmeBase, domainID, recordID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	return d.doJSON(req, nil)
}
