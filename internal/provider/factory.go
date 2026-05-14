package provider

import "fmt"

// New creates a Provider from a provider name and credentials map.
func New(name string, creds map[string]string) (Provider, error) {
	switch name {
	case "cloudflare":
		token, ok := creds["api_token"]
		if !ok {
			return nil, fmt.Errorf("cloudflare: missing api_token credential")
		}
		return NewCloudflare(token), nil
	case "godaddy":
		key, ok := creds["api_key"]
		if !ok {
			return nil, fmt.Errorf("godaddy: missing api_key credential")
		}
		secret, ok := creds["api_secret"]
		if !ok {
			return nil, fmt.Errorf("godaddy: missing api_secret credential")
		}
		return NewGoDaddy(key, secret), nil
	case "dnsmadeeasy":
		key, ok := creds["api_key"]
		if !ok {
			return nil, fmt.Errorf("dnsmadeeasy: missing api_key credential")
		}
		secret, ok := creds["api_secret"]
		if !ok {
			return nil, fmt.Errorf("dnsmadeeasy: missing api_secret credential")
		}
		return NewDNSMadeEasy(key, secret), nil
	case "test":
		return NewTest(), nil
	default:
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
}
