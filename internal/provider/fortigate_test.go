package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEntryToRecord(t *testing.T) {
	cases := []struct {
		name string
		in   fgEntry
		want Record
	}{
		{
			name: "A",
			in:   fgEntry{ID: 1, Type: "A", Hostname: "www", IP: "1.2.3.4", TTL: 300},
			want: Record{ID: "1", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300},
		},
		{
			name: "AAAA",
			in:   fgEntry{ID: 2, Type: "AAAA", Hostname: "www", IPv6: "::1", TTL: 300},
			want: Record{ID: "2", Type: "AAAA", Name: "www", Value: "::1", TTL: 300},
		},
		{
			name: "CNAME",
			in:   fgEntry{ID: 3, Type: "CNAME", Hostname: "blog", CanonicalName: "www.example.com", TTL: 300},
			want: Record{ID: "3", Type: "CNAME", Name: "blog", Value: "www.example.com", TTL: 300},
		},
		{
			name: "MX",
			in:   fgEntry{ID: 4, Type: "MX", Hostname: "mail.example.com", Preference: 10, TTL: 3600},
			want: Record{ID: "4", Type: "MX", Name: "@", Value: "mail.example.com", TTL: 3600, Priority: 10},
		},
		{
			name: "NS",
			in:   fgEntry{ID: 5, Type: "NS", Hostname: "ns1.example.com", TTL: 3600},
			want: Record{ID: "5", Type: "NS", Name: "@", Value: "ns1.example.com", TTL: 3600},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := entryToRecord(tc.in)
			if got != tc.want {
				t.Errorf("entryToRecord(%v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRecordToEntry(t *testing.T) {
	cases := []struct {
		name      string
		in        Record
		wantField string
		wantValue interface{}
	}{
		{"A", Record{Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300}, "ip", "1.2.3.4"},
		{"AAAA", Record{Type: "AAAA", Name: "www", Value: "::1", TTL: 300}, "ipv6", "::1"},
		{"CNAME", Record{Type: "CNAME", Name: "blog", Value: "www.example.com", TTL: 300}, "canonical-name", "www.example.com"},
		{"MX target stored in hostname", Record{Type: "MX", Value: "mail.example.com", Priority: 10, TTL: 3600}, "hostname", "mail.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := recordToEntry(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := entry[tc.wantField]; got != tc.wantValue {
				t.Errorf("entry[%q] = %v, want %v", tc.wantField, got, tc.wantValue)
			}
		})
	}
}

func TestRecordToEntry_MXPreference(t *testing.T) {
	entry, err := recordToEntry(Record{Type: "MX", Value: "mail.example.com", Priority: 20, TTL: 3600})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry["preference"] != 20 {
		t.Errorf("expected preference=20, got %v", entry["preference"])
	}
}

func TestRecordToEntry_UnsupportedType(t *testing.T) {
	if _, err := recordToEntry(Record{Type: "CAA", Name: "@", Value: "0 issue \"letsencrypt.org\""}); err == nil {
		t.Fatal("expected error for unsupported CAA record type")
	}
}

func TestRecordToEntry_InvalidID(t *testing.T) {
	if _, err := recordToEntry(Record{ID: "not-a-number", Type: "A", Name: "www", Value: "1.2.3.4"}); err == nil {
		t.Fatal("expected error for non-numeric record ID")
	}
}

// TestFortiGate_HTTPRoundTrip exercises the URL/header/payload shape
// against a stub HTTP server. It catches regressions in vdom handling,
// auth, and the FortiOS response envelope.
func TestFortiGate_HTTPRoundTrip(t *testing.T) {
	var captured struct {
		method string
		path   string
		query  string
		auth   string
		body   string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.query = r.URL.RawQuery
		captured.auth = r.Header.Get("Authorization")
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			captured.body = string(b)
		}

		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/cmdb/system/dns-database":
			_ = json.NewEncoder(w).Encode(fgListResponse{
				HTTPStatus: 200,
				Results: []fgZone{
					{Name: "internal", Domain: "example.com"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/api/v2/cmdb/system/dns-database/internal":
			_ = json.NewEncoder(w).Encode(fgListResponse{
				HTTPStatus: 200,
				Results: []fgZone{{
					Name:   "internal",
					Domain: "example.com",
					DNSEntry: []fgEntry{
						{ID: 1, Type: "A", Hostname: "www", IP: "1.2.3.4", TTL: 300},
					},
				}},
			})
		case r.Method == "POST":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	f := NewFortiGate(srv.URL, "tok", "vdom1", false)
	ctx := context.Background()

	domains, err := f.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(domains) != 1 || domains[0].ID != "internal" || domains[0].Name != "example.com" {
		t.Fatalf("unexpected domains: %+v", domains)
	}
	if captured.auth != "Bearer tok" {
		t.Errorf("expected Bearer auth header, got %q", captured.auth)
	}
	if !strings.Contains(captured.query, "vdom=vdom1") {
		t.Errorf("expected vdom query param, got %q", captured.query)
	}

	records, err := f.ListRecords(ctx, "internal")
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 1 || records[0].Type != "A" || records[0].Value != "1.2.3.4" {
		t.Fatalf("unexpected records: %+v", records)
	}

	if err := f.CreateRecord(ctx, "internal", Record{Type: "A", Name: "api", Value: "5.6.7.8", TTL: 60}); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if captured.method != "POST" {
		t.Errorf("expected POST, got %s", captured.method)
	}
	if !strings.Contains(captured.body, `"ip":"5.6.7.8"`) {
		t.Errorf("expected ip in body, got %q", captured.body)
	}
	if !strings.Contains(captured.path, "/dns-entry") {
		t.Errorf("expected /dns-entry in path, got %q", captured.path)
	}
}

func TestFortiGate_DefaultVDOM(t *testing.T) {
	f := NewFortiGate("https://example.com", "tok", "", false)
	if f.vdom != "root" {
		t.Errorf("expected default vdom 'root', got %q", f.vdom)
	}
}
