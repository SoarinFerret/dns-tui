package provider

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

// Test is an in-memory provider for testing. All operations are
// synchronous and require no network access.
type Test struct {
	mu      sync.Mutex
	domains []Domain
	records map[string][]Record // domainID -> records
	nextID  int
}

// NewTest creates a Test provider pre-populated with the given domains.
// Use AddDomain and AddRecord to set up state before exercising code
// that calls the Provider interface.
func NewTest() *Test {
	return &Test{
		records: make(map[string][]Record),
	}
}

// AddDomain adds a domain and returns its ID.
func (t *Test) AddDomain(name string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := t.genID()
	t.domains = append(t.domains, Domain{ID: id, Name: name})
	return id
}

// AddRecord adds a record under the given domain ID and returns the record ID.
func (t *Test) AddRecord(domainID string, r Record) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	r.ID = t.genID()
	t.records[domainID] = append(t.records[domainID], r)
	return r.ID
}

func (t *Test) genID() string {
	t.nextID++
	return strconv.Itoa(t.nextID)
}

func (t *Test) ListDomains(_ context.Context) ([]Domain, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Domain, len(t.domains))
	copy(out, t.domains)
	return out, nil
}

func (t *Test) ListRecords(_ context.Context, domainID string) ([]Record, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	recs, ok := t.records[domainID]
	if !ok {
		return nil, nil
	}
	out := make([]Record, len(recs))
	copy(out, recs)
	return out, nil
}

func (t *Test) CreateRecord(_ context.Context, domainID string, r Record) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	r.ID = t.genID()
	t.records[domainID] = append(t.records[domainID], r)
	return nil
}

func (t *Test) UpdateRecord(_ context.Context, domainID string, r Record) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	recs := t.records[domainID]
	for i, existing := range recs {
		if existing.ID == r.ID {
			recs[i] = r
			return nil
		}
	}
	return fmt.Errorf("test: record %q not found in domain %q", r.ID, domainID)
}

func (t *Test) DeleteRecord(_ context.Context, domainID string, recordID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	recs := t.records[domainID]
	for i, existing := range recs {
		if existing.ID == recordID {
			t.records[domainID] = append(recs[:i], recs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("test: record %q not found in domain %q", recordID, domainID)
}
