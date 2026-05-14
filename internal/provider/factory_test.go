package provider_test

import (
	"testing"

	"github.com/soarinferret/dns-tui/internal/provider"
)

func TestNew_Cloudflare(t *testing.T) {
	p, err := provider.New("cloudflare", map[string]string{"api_token": "tok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_CloudflareMissingToken(t *testing.T) {
	_, err := provider.New("cloudflare", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing api_token")
	}
}

func TestNew_GoDaddy(t *testing.T) {
	p, err := provider.New("godaddy", map[string]string{"api_key": "k", "api_secret": "s"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_GoDaddyMissingKey(t *testing.T) {
	_, err := provider.New("godaddy", map[string]string{"api_secret": "s"})
	if err == nil {
		t.Fatal("expected error for missing api_key")
	}
}

func TestNew_GoDaddyMissingSecret(t *testing.T) {
	_, err := provider.New("godaddy", map[string]string{"api_key": "k"})
	if err == nil {
		t.Fatal("expected error for missing api_secret")
	}
}

func TestNew_DNSMadeEasy(t *testing.T) {
	p, err := provider.New("dnsmadeeasy", map[string]string{"api_key": "k", "api_secret": "s"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_DNSMadeEasyMissingKey(t *testing.T) {
	_, err := provider.New("dnsmadeeasy", map[string]string{"api_secret": "s"})
	if err == nil {
		t.Fatal("expected error for missing api_key")
	}
}

func TestNew_DNSMadeEasyMissingSecret(t *testing.T) {
	_, err := provider.New("dnsmadeeasy", map[string]string{"api_key": "k"})
	if err == nil {
		t.Fatal("expected error for missing api_secret")
	}
}

func TestNew_FortiGate(t *testing.T) {
	p, err := provider.New("fortigate", map[string]string{
		"host":      "https://fortigate.example.com",
		"api_token": "tok",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_FortiGateMissingHost(t *testing.T) {
	_, err := provider.New("fortigate", map[string]string{"api_token": "tok"})
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

func TestNew_FortiGateMissingToken(t *testing.T) {
	_, err := provider.New("fortigate", map[string]string{"host": "https://fortigate.example.com"})
	if err == nil {
		t.Fatal("expected error for missing api_token")
	}
}

func TestNew_TestProvider(t *testing.T) {
	p, err := provider.New("test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_Unknown(t *testing.T) {
	_, err := provider.New("bogus", nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
