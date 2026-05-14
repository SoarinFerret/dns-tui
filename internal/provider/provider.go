package provider

import "context"

// Domain represents a DNS zone/domain.
type Domain struct {
	ID   string
	Name string
}

// Record represents a single DNS record.
type Record struct {
	ID       string
	Type     string
	Name     string
	Value    string
	TTL      int
	Priority int // MX/SRV only
}

// Provider is the interface every DNS provider must implement.
type Provider interface {
	ListDomains(ctx context.Context) ([]Domain, error)
	ListRecords(ctx context.Context, domainID string) ([]Record, error)
	CreateRecord(ctx context.Context, domainID string, r Record) error
	UpdateRecord(ctx context.Context, domainID string, r Record) error
	DeleteRecord(ctx context.Context, domainID string, recordID string) error
}
