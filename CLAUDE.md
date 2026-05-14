# dns-tui

A terminal UI application written in Go for managing DNS records across multiple cloud providers.

## Development

All development is done inside a Nix dev shell. The `flake.nix` should provide the dev shell and also allow importing as a package on NixOS systems.

```bash
# Enter dev shell
nix develop

# Build
go build ./cmd/dns-tui

# Run
go run ./cmd/dns-tui

# Test
go test ./...

# Lint
golangci-lint run
```

Module path: `github.com/soarinferret/dns-tui`

## Project Structure

```
cmd/dns-tui/          # Application entrypoint
internal/
  config/             # YAML config loading/saving
  provider/           # Provider interface and implementations
    provider.go       # Interface + shared types
    cloudflare.go
    godaddy.go
    dnsmadeeasy.go
  tui/                # tview UI components
    app.go            # Main application, layout assembly
    profiles.go       # Left sidebar — profile list
    domains.go        # Middle sidebar — domain list
    records.go        # Main pane — record table with pagination
    modals.go         # Add/edit/delete confirmation dialogs
    statusbar.go      # Bottom status bar for errors and info
```

## Architecture

```
Config (YAML) → Provider Factory → Provider Interface → TUI
```

1. On startup, load `~/.config/dns-tui/config.yaml` to get the list of profiles
2. The profiles pane displays all configured profiles
3. When a profile is selected, instantiate the appropriate provider using a factory function
4. The provider fetches domains, which populate the domains pane
5. When a domain is selected, the provider fetches records, which populate the records pane
6. All CRUD operations on records go through the provider interface

## Provider Interface

```go
// Domain represents a DNS zone/domain
type Domain struct {
    ID   string
    Name string // e.g. "example.com"
}

// Record represents a single DNS record
type Record struct {
    ID       string
    Type     string // A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
    Name     string // e.g. "www"
    Value    string // e.g. "192.168.1.1"
    TTL      int    // seconds
    Priority int    // MX/SRV only
}

// Provider is the interface every DNS provider must implement
type Provider interface {
    ListDomains(ctx context.Context) ([]Domain, error)
    ListRecords(ctx context.Context, domainID string) ([]Record, error)
    CreateRecord(ctx context.Context, domainID string, r Record) error
    UpdateRecord(ctx context.Context, domainID string, r Record) error
    DeleteRecord(ctx context.Context, domainID string, recordID string) error
}
```

Supported providers:
- **Cloudflare** — uses API token auth (`Authorization: Bearer <token>`)
- **GoDaddy** — uses API key + secret
- **DNSMadeEasy** — uses API key + secret with HMAC request signing

Use `net/http` directly for provider API calls rather than pulling in provider-specific SDKs.

## Supported Record Types

A, AAAA, CNAME, MX, TXT, NS, SRV, CAA

When adding/editing a record, the form fields should adapt to the record type (e.g. show Priority for MX/SRV, show weight/port/target for SRV).

## Configuration

Location: `~/.config/dns-tui/config.yaml`

```yaml
profiles:
  - name: "Cloudflare Production"
    provider: cloudflare
    credentials:
      api_token: "cf-token-here"

  - name: "GoDaddy Personal"
    provider: godaddy
    credentials:
      api_key: "key-here"
      api_secret: "secret-here"

  - name: "DNSMadeEasy"
    provider: dnsmadeeasy
    credentials:
      api_key: "key-here"
      api_secret: "secret-here"
```

## TUI Layout

Built with [tview](https://github.com/rivo/tview).

```
+----------------+--------------------+------------------------------------+
|   Profiles     |     Domains        |           Records                  |
|   (width: 20%) |     (width: 20%)   |           (width: 60%)             |
|                |                    |                                    |
|  > CF Prod     |  > example.com     |  Type  Name     Value       TTL   |
|    GoDaddy     |    other.dev       |  A     @        1.2.3.4     300   |
|    DME         |    mysite.org      |  A     www      1.2.3.4     300   |
|                |                    |  CNAME blog     example.com 3600  |
|                |                    |  MX    @        mail.ex.com 3600  |
|                |                    |  TXT   @        v=spf1...   3600  |
|                |                    |                                    |
|                |                    |  Page 1/3        [n]ext [p]rev    |
+----------------+--------------------+------------------------------------+
| Status: Ready                                                            |
+--------------------------------------------------------------------------+
```

### Focus behavior

- **Tab** / **Shift-Tab** — cycle focus between the three panes (Profiles → Domains → Records → Profiles)
- The focused pane gets a highlighted border
- Selection in a pane triggers a cascade: selecting a profile loads its domains, selecting a domain loads its records

### Keybindings

**Global:**
- `Tab` / `Shift-Tab` — move focus between panes
- `q` — quit
- `?` — show help modal

**Navigation (in any list/table):**
- `j` / `Down` — move down
- `k` / `Up` — move up
- `Enter` — select / confirm

**Records pane:**
- `a` — add new record (opens form modal)
- `e` — edit selected record (opens form modal pre-filled)
- `d` — delete selected record (opens confirmation modal)
- `/` — search/filter records by name or value
- `Esc` — clear search filter
- `n` — next page
- `p` — previous page

## Pagination

Records are displayed in pages of 50. The status area at the bottom of the records pane shows the current page and total pages. Use `n`/`p` to navigate.

## Error Handling

- API errors display in the status bar at the bottom of the screen with the error message
- The status bar auto-clears after 5 seconds or on the next successful operation
- Destructive operations (delete) always show a confirmation modal before proceeding
- Network timeouts: 10 second default timeout on all provider API calls

## Dependencies

- [tview](https://github.com/rivo/tview) — terminal UI framework
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) — YAML config parsing
