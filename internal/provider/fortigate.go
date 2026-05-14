package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// FortiGate implements Provider using the FortiOS REST API
// (/api/v2/cmdb/system/dns-database). The "domain" abstraction maps to
// a DNS database zone and records map to entries within that zone.
type FortiGate struct {
	host   string // e.g. "https://fortigate.example.com"
	token  string
	vdom   string
	client *http.Client
}

func NewFortiGate(host, token, vdom string, insecure bool) *FortiGate {
	host = strings.TrimRight(host, "/")
	if vdom == "" {
		vdom = "root"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &FortiGate{
		host:  host,
		token: token,
		vdom:  vdom,
		client: &http.Client{
			Timeout:   apiTimeout,
			Transport: transport,
		},
	}
}

const fgPath = "/api/v2/cmdb/system/dns-database"

type fgZone struct {
	Name     string    `json:"name"`
	Domain   string    `json:"domain"`
	DNSEntry []fgEntry `json:"dns-entry"`
}

type fgEntry struct {
	ID            int    `json:"id"`
	Status        string `json:"status,omitempty"`
	Type          string `json:"type"`
	TTL           int    `json:"ttl"`
	Preference    int    `json:"preference,omitempty"`
	IP            string `json:"ip,omitempty"`
	IPv6          string `json:"ipv6,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	CanonicalName string `json:"canonical-name,omitempty"`
}

type fgListResponse struct {
	HTTPStatus int      `json:"http_status"`
	Status     string   `json:"status"`
	Reason     string   `json:"reason"`
	Results    []fgZone `json:"results"`
}

func (f *FortiGate) url(path string) string {
	u := f.host + path
	q := url.Values{}
	q.Set("vdom", f.vdom)
	return u + "?" + q.Encode()
}

func (f *FortiGate) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return f.client.Do(req)
}

// doJSON performs a request and decodes the JSON body into v. If the
// response indicates an error, it returns a formatted error containing
// the FortiGate reason.
func (f *FortiGate) doJSON(req *http.Request, v interface{}) error {
	resp, err := f.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		var errBody struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		reason := errBody.Reason
		if reason == "" {
			reason = resp.Status
		}
		return fmt.Errorf("fortigate: %s", reason)
	}
	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

func (f *FortiGate) ListDomains(ctx context.Context) ([]Domain, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", f.url(fgPath), nil)
	if err != nil {
		return nil, err
	}
	var resp fgListResponse
	if err := f.doJSON(req, &resp); err != nil {
		return nil, err
	}
	domains := make([]Domain, len(resp.Results))
	for i, z := range resp.Results {
		name := z.Domain
		if name == "" {
			name = z.Name
		}
		domains[i] = Domain{ID: z.Name, Name: name}
	}
	return domains, nil
}

func (f *FortiGate) ListRecords(ctx context.Context, domainID string) ([]Record, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", f.url(fgPath+"/"+url.PathEscape(domainID)), nil)
	if err != nil {
		return nil, err
	}
	var resp fgListResponse
	if err := f.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 {
		return nil, nil
	}
	entries := resp.Results[0].DNSEntry
	records := make([]Record, 0, len(entries))
	for _, e := range entries {
		records = append(records, entryToRecord(e))
	}
	return records, nil
}

func (f *FortiGate) CreateRecord(ctx context.Context, domainID string, r Record) error {
	entry, err := recordToEntry(r)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(entry)
	endpoint := f.url(fgPath + "/" + url.PathEscape(domainID) + "/dns-entry")
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	return f.doJSON(req, nil)
}

func (f *FortiGate) UpdateRecord(ctx context.Context, domainID string, r Record) error {
	entry, err := recordToEntry(r)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(entry)
	endpoint := f.url(fgPath + "/" + url.PathEscape(domainID) + "/dns-entry/" + url.PathEscape(r.ID))
	req, err := http.NewRequestWithContext(ctx, "PUT", endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	return f.doJSON(req, nil)
}

func (f *FortiGate) DeleteRecord(ctx context.Context, domainID string, recordID string) error {
	endpoint := f.url(fgPath + "/" + url.PathEscape(domainID) + "/dns-entry/" + url.PathEscape(recordID))
	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	return f.doJSON(req, nil)
}

// entryToRecord converts a FortiGate dns-entry to the generic Record
// representation. FortiGate stores the "value" of a record in different
// fields depending on the record type.
func entryToRecord(e fgEntry) Record {
	r := Record{
		ID:   strconv.Itoa(e.ID),
		Type: e.Type,
		Name: e.Hostname,
		TTL:  e.TTL,
	}
	switch strings.ToUpper(e.Type) {
	case "A":
		r.Value = e.IP
	case "AAAA":
		r.Value = e.IPv6
	case "CNAME":
		r.Value = e.CanonicalName
	case "MX":
		// FortiGate stores MX target in hostname; the record applies
		// to the zone apex. Surface as name="@" for clarity.
		r.Name = "@"
		r.Value = e.Hostname
		r.Priority = e.Preference
	case "NS":
		r.Name = "@"
		r.Value = e.Hostname
	default:
		// PTR / SRV / TXT: best-effort, treat hostname as value.
		r.Value = e.Hostname
	}
	return r
}

// recordToEntry maps a generic Record onto the FortiGate dns-entry
// payload shape. Returns an error for record types FortiGate's
// dns-database API cannot represent.
func recordToEntry(r Record) (map[string]interface{}, error) {
	t := strings.ToUpper(r.Type)
	entry := map[string]interface{}{
		"type":   t,
		"ttl":    r.TTL,
		"status": "enable",
	}
	if r.ID != "" {
		id, err := strconv.Atoi(r.ID)
		if err != nil {
			return nil, fmt.Errorf("fortigate: invalid record ID %q", r.ID)
		}
		entry["id"] = id
	}
	switch t {
	case "A":
		entry["hostname"] = r.Name
		entry["ip"] = r.Value
	case "AAAA":
		entry["hostname"] = r.Name
		entry["ipv6"] = r.Value
	case "CNAME":
		entry["hostname"] = r.Name
		entry["canonical-name"] = r.Value
	case "MX":
		entry["hostname"] = r.Value
		entry["preference"] = r.Priority
	case "NS":
		entry["hostname"] = r.Value
	case "PTR":
		entry["hostname"] = r.Value
	case "":
		return nil, fmt.Errorf("fortigate: record type is required")
	default:
		return nil, fmt.Errorf("fortigate: record type %q not supported by /system/dns-database", r.Type)
	}
	return entry, nil
}
