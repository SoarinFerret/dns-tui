package provider_test

import (
	"context"
	"testing"

	"github.com/soarinferret/dns-tui/internal/provider"
)

func seedProvider(t *testing.T) (*provider.Test, string) {
	t.Helper()
	p := provider.NewTest()
	domID := p.AddDomain("example.com")
	p.AddRecord(domID, provider.Record{Type: "A", Name: "@", Value: "1.2.3.4", TTL: 300})
	p.AddRecord(domID, provider.Record{Type: "CNAME", Name: "www", Value: "example.com", TTL: 3600})
	p.AddRecord(domID, provider.Record{Type: "MX", Name: "@", Value: "mail.example.com", TTL: 3600, Priority: 10})
	return p, domID
}

func TestListDomains(t *testing.T) {
	p := provider.NewTest()
	p.AddDomain("example.com")
	p.AddDomain("other.dev")

	domains, err := p.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0].Name != "example.com" {
		t.Errorf("expected first domain example.com, got %s", domains[0].Name)
	}
}

func TestListRecords(t *testing.T) {
	p, domID := seedProvider(t)

	records, err := p.ListRecords(context.Background(), domID)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
}

func TestListRecords_EmptyDomain(t *testing.T) {
	p := provider.NewTest()
	domID := p.AddDomain("empty.com")

	records, err := p.ListRecords(context.Background(), domID)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestListRecords_UnknownDomain(t *testing.T) {
	p := provider.NewTest()

	records, err := p.ListRecords(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records, got %v", records)
	}
}

func TestCreateRecord(t *testing.T) {
	p, domID := seedProvider(t)

	err := p.CreateRecord(context.Background(), domID, provider.Record{
		Type:  "TXT",
		Name:  "@",
		Value: "v=spf1 include:_spf.google.com ~all",
		TTL:   3600,
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	records, _ := p.ListRecords(context.Background(), domID)
	if len(records) != 4 {
		t.Fatalf("expected 4 records after create, got %d", len(records))
	}

	last := records[len(records)-1]
	if last.Type != "TXT" || last.Name != "@" {
		t.Errorf("unexpected record: %+v", last)
	}
	if last.ID == "" {
		t.Error("created record should have an ID")
	}
}

func TestUpdateRecord(t *testing.T) {
	p, domID := seedProvider(t)

	records, _ := p.ListRecords(context.Background(), domID)
	rec := records[0] // the A record
	rec.Value = "5.6.7.8"

	err := p.UpdateRecord(context.Background(), domID, rec)
	if err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}

	updated, _ := p.ListRecords(context.Background(), domID)
	if updated[0].Value != "5.6.7.8" {
		t.Errorf("expected updated value 5.6.7.8, got %s", updated[0].Value)
	}
}

func TestUpdateRecord_NotFound(t *testing.T) {
	p, domID := seedProvider(t)

	err := p.UpdateRecord(context.Background(), domID, provider.Record{ID: "999"})
	if err == nil {
		t.Fatal("expected error for nonexistent record")
	}
}

func TestDeleteRecord(t *testing.T) {
	p, domID := seedProvider(t)

	records, _ := p.ListRecords(context.Background(), domID)
	err := p.DeleteRecord(context.Background(), domID, records[0].ID)
	if err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	remaining, _ := p.ListRecords(context.Background(), domID)
	if len(remaining) != 2 {
		t.Fatalf("expected 2 records after delete, got %d", len(remaining))
	}
	for _, r := range remaining {
		if r.ID == records[0].ID {
			t.Error("deleted record still present")
		}
	}
}

func TestDeleteRecord_NotFound(t *testing.T) {
	p, domID := seedProvider(t)

	err := p.DeleteRecord(context.Background(), domID, "999")
	if err == nil {
		t.Fatal("expected error for nonexistent record")
	}
}

// Verify Test satisfies the Provider interface at compile time.
var _ provider.Provider = (*provider.Test)(nil)
