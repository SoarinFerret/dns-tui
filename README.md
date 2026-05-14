# dns-tui

> **Built with [Claude](https://claude.com/claude-code).** This project was written end-to-end by Claude Code. Review the source before trusting it with production DNS credentials.

A terminal UI for managing DNS records across multiple cloud providers, written in Go using [tview](https://github.com/rivo/tview).

## Features

- Three-pane layout: profiles, domains, records
- Multiple provider support: **Cloudflare**, **GoDaddy**, **DNSMadeEasy**, **FortiGate** (built-in DNS server)
- Multiple profiles per provider (e.g. separate prod and personal Cloudflare accounts)
- Full CRUD on records: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV`, `CAA`
- Form fields adapt to the selected record type (priority for `MX`/`SRV`, weight/port/target for `SRV`)
- Search/filter records by name or value
- Pagination for large zones (50 records per page)
- Vim-style navigation (`j`/`k`) and arrow keys

## Install

### From source

```bash
go install github.com/soarinferret/dns-tui/cmd/dns-tui@latest
```

### Nix

A `flake.nix` is provided. Build with:

```bash
nix build
./result/bin/dns-tui
```

Or run directly:

```bash
nix run github:soarinferret/dns-tui
```

### Build manually

```bash
git clone https://github.com/soarinferret/dns-tui
cd dns-tui
go build -o dns-tui ./cmd/dns-tui
```

## Configuration

dns-tui reads `~/.config/dns-tui/config.yaml` on startup. If non-existent, you will be prompted to setup a provider.

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

  - name: "FortiGate Home"
    provider: fortigate
    credentials:
      host: "https://fortigate.example.com"
      api_token: "fg-token-here"
      vdom: "root"                  # optional, defaults to "root"
      insecure_skip_verify: "true"  # optional, for self-signed certs
```

### Credential requirements

| Provider     | Required fields              | Notes                                         |
|--------------|------------------------------|-----------------------------------------------|
| Cloudflare   | `api_token`                  | Scoped API token with Zone:Read / DNS:Edit    |
| GoDaddy      | `api_key`, `api_secret`      | Production keys from developer.godaddy.com    |
| DNSMadeEasy  | `api_key`, `api_secret`      | Used for HMAC request signing                 |
| FortiGate    | `host`, `api_token`          | Optional `vdom` (default `root`) and `insecure_skip_verify` for self-signed certs. Uses `/api/v2/cmdb/system/dns-database`. TXT/SRV/CAA not supported by the FortiOS schema. |

The config file holds plaintext credentials. Protect it with `chmod 600 ~/.config/dns-tui/config.yaml`.

## Usage

```bash
dns-tui
```

### Keybindings

**Global**

| Key              | Action                       |
|------------------|------------------------------|
| `Tab` / `S-Tab`  | Cycle focus between panes    |
| `?`              | Help modal                   |
| `q`              | Quit                         |

**Navigation**

| Key              | Action          |
|------------------|-----------------|
| `j` / `↓`        | Move down       |
| `k` / `↑`        | Move up         |
| `Enter`          | Select          |

**Records pane**

| Key  | Action                                   |
|------|------------------------------------------|
| `a`  | Add a new record                         |
| `e`  | Edit the selected record                 |
| `d`  | Delete the selected record (confirms)    |
| `/`  | Search/filter by name or value           |
| `Esc`| Clear search filter                      |
| `n`  | Next page                                |
| `p`  | Previous page                            |

## Development

All development happens inside the Nix dev shell:

```bash
nix develop
```

Common tasks via [just](https://github.com/casey/just):

```bash
just build      # go build -o dns-tui ./cmd/dns-tui
just run        # go run ./cmd/dns-tui
just test       # go test ./...
just lint       # golangci-lint run
just fmt        # gofmt -w .
just tidy       # go mod tidy
```

### Project layout

```
cmd/dns-tui/          Application entrypoint
internal/
  config/             YAML config loading
  provider/           Provider interface and implementations
    cloudflare.go
    godaddy.go
    dnsmadeeasy.go
    fortigate.go
  tui/                tview UI components
```

### Adding a new provider

1. Implement the `provider.Provider` interface in `internal/provider/<name>.go`
2. Register a constructor in `internal/provider/factory.go`
3. Document the credential fields in this README

Provider calls use `net/http` directly — no vendor SDKs. All requests have a 10-second timeout.

## License

[MIT](LICENSE)
