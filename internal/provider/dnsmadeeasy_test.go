package provider

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"testing"
	"time"
)

func TestDNSMadeEasySign(t *testing.T) {
	d := NewDNSMadeEasy("test-api-key", "test-secret")
	req, _ := http.NewRequest("GET", "https://api.dnsmadeeasy.com/V2.0/dns/managed", nil)
	d.sign(req)

	// Headers must use exact casing (not Go's canonical form).
	// Access the map directly to verify the keys weren't canonicalized.
	//nolint:staticcheck // intentionally checking non-canonical keys
	apiKeyVals, ok := req.Header["x-dnsme-apiKey"]
	if !ok || len(apiKeyVals) == 0 {
		t.Fatal("x-dnsme-apiKey header missing (may have been canonicalized to X-Dnsme-Apikey)")
	}
	if apiKeyVals[0] != "test-api-key" {
		t.Errorf("expected x-dnsme-apiKey 'test-api-key', got %q", apiKeyVals[0])
	}

	//nolint:staticcheck // intentionally checking non-canonical keys
	dateVals, ok := req.Header["x-dnsme-requestDate"]
	if !ok || len(dateVals) == 0 {
		t.Fatal("x-dnsme-requestDate header missing (may have been canonicalized)")
	}
	dateStr := dateVals[0]

	// Date must end with "GMT", not "UTC" or "+0000"
	if len(dateStr) < 3 || dateStr[len(dateStr)-3:] != "GMT" {
		t.Errorf("x-dnsme-requestDate should end with GMT, got %q", dateStr)
	}

	// Date must be parseable as HTTP-date
	if _, err := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", dateStr); err != nil {
		t.Errorf("x-dnsme-requestDate not valid HTTP-date: %v", err)
	}

	// HMAC must match what we compute from the date header
	mac := hmac.New(sha1.New, []byte("test-secret"))
	mac.Write([]byte(dateStr))
	expected := hex.EncodeToString(mac.Sum(nil))
	//nolint:staticcheck // intentionally checking non-canonical key
	hmacVals := req.Header["x-dnsme-hmac"]
	if len(hmacVals) == 0 {
		t.Fatal("x-dnsme-hmac header missing (may have been canonicalized)")
	}
	if hmacVals[0] != expected {
		t.Errorf("HMAC mismatch: expected %s, got %s", expected, hmacVals[0])
	}
}
