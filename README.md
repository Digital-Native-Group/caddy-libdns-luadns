Lua DNS module for Caddy
===========================

This package contains a DNS provider module for [Caddy](https://github.com/caddyserver/caddy). It can be used to manage DNS records with [Lua DNS](https://www.luadns.com/).

## Implementation Summary

This module implements complete Lua DNS support for Caddy through a two-layer architecture:

### libdns Provider Layer (`libdns-luadns/`)
The core DNS operations library implementing the [libdns interfaces](https://github.com/libdns/libdns):

- **Full libdns Interface**: Implements all four record management interfaces:
  - `GetRecords()` - Retrieve all DNS records for a zone
  - `AppendRecords()` - Create new DNS records
  - `SetRecords()` - Update existing records or create if not found
  - `DeleteRecords()` - Remove DNS records

- **Direct API Implementation**: Built from scratch using the [Lua DNS REST API](https://www.luadns.com/api.html) without depending on external wrappers, resulting in clean, maintainable code.

- **HTTP Basic Authentication**: Uses email + API key credentials as per Lua DNS API requirements.

- **Zone ID Caching**: Automatically caches zone name to zone ID mappings with mutex-protected concurrent access, minimizing API calls and improving performance.

- **Proper Error Handling**: Returns structured errors from API responses with appropriate context.

- **Rate Limit Awareness**: Designed with Lua DNS's rate limits in mind (1200 requests per 5 minutes).

### Caddy Integration Layer (`module.go`)
The Caddy module wrapper that connects libdns to Caddy's configuration system:

- **Module ID**: `dns.providers.luadns`

- **Flexible Configuration**: Supports multiple Caddyfile syntax styles:
  - Inline: `luadns email api_key`
  - Block format with `email` and `api_key` directives
  - JSON configuration for programmatic setups

- **Environment Variable Support**: Uses Caddy's replacer to support `{env.VARIABLE}` placeholders for secure credential management.

- **Proper Provisioning**: Implements `caddy.Provisioner` interface for initialization during Caddy startup.

- **Caddyfile Parsing**: Implements `caddyfile.Unmarshaler` for human-readable configuration.

### Project Structure
```
.
├── libdns-luadns/          # libdns provider (can be published to github.com/libdns/luadns)
│   ├── provider.go         # libdns interface implementation
│   ├── client.go           # HTTP client for Lua DNS API
│   ├── go.mod              # Independent module definition
│   └── README.md           # Provider-specific documentation
├── module.go               # Caddy module integration
├── go.mod                  # Caddy module (uses replace directive for local dev)
└── README.md               # This file
```

The implementation follows Caddy ecosystem best practices and can be split into two separate repositories when ready for publication.

## Caddy module name

```
dns.providers.luadns
```

## Authenticating

To authenticate with Lua DNS, you need:
1. Your account email address
2. An API key from https://www.luadns.com/api_keys

API keys can be configured as global (access to all zones) or zone-restricted for additional security.

## Config examples

To use this module for the ACME DNS challenge, [configure the ACME issuer in your Caddy JSON](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/) like so:

```json
{
	"module": "acme",
	"challenges": {
		"dns": {
			"provider": {
				"name": "luadns",
				"email": "your@email.com",
				"api_key": "{env.LUADNS_API_KEY}"
			}
		}
	}
}
```

or with the Caddyfile:

```
# Globally using environment variables (recommended)
{
	acme_dns luadns {
		email your@email.com
		api_key {env.LUADNS_API_KEY}
	}
}
```

```
# Per-site with inline credentials
example.com {
	tls {
		dns luadns your@email.com your-api-key
	}
}
```

```
# Per-site with block format
example.com {
	tls {
		dns luadns {
			email your@email.com
			api_key {env.LUADNS_API_KEY}
		}
	}
}
```

## Building

To compile Caddy with this module:

```bash
xcaddy build --with github.com/caddy-dns/luadns
```

Or to build locally during development:

```bash
xcaddy build --with github.com/caddy-dns/luadns=./
```

## Development Notes

### Current Setup
The project currently uses a Go module `replace` directive to reference the local `libdns-luadns` directory:

```go
replace github.com/libdns/luadns => ./libdns-luadns
```

This allows for integrated development and testing before splitting into separate repositories.

### Publishing Checklist
When ready to publish:

1. **Publish libdns provider first**:
   - Move `libdns-luadns/` to a new repository at `github.com/libdns/luadns`
   - Tag with semantic version (e.g., `v1.0.0`)
   - Update the module path in `libdns-luadns/go.mod` if needed

2. **Update Caddy module**:
   - Remove the `replace` directive from `go.mod`
   - Run `go get github.com/libdns/luadns@latest`
   - Update `go.mod` to reference the published version
   - Update repository URL to `github.com/caddy-dns/luadns`

3. **Finalize documentation**:
   - Update LICENSE with your name and current year
   - Ensure all examples use production repository paths
   - Add badges for Go Report Card, Go Reference, etc.

4. **Optional enhancements**:
   - Add unit tests for provider methods
   - Add integration tests with Lua DNS API
   - Implement rate limit backoff strategy
   - Add CI/CD workflows for automated testing

### Testing
To test the module with actual Lua DNS credentials:

```bash
# Build Caddy with the module
xcaddy build --with github.com/caddy-dns/luadns=./

# Create a Caddyfile for testing
cat > Caddyfile <<EOF
{
  acme_dns luadns {
    email {env.LUADNS_EMAIL}
    api_key {env.LUADNS_API_KEY}
  }
}

example.com {
  tls {
    dns luadns
  }
}
EOF

# Run with your credentials
export LUADNS_EMAIL="your@email.com"
export LUADNS_API_KEY="your-api-key"
./caddy run
```

## API Reference

- **Lua DNS API Documentation**: https://www.luadns.com/api.html
- **libdns Interface Documentation**: https://pkg.go.dev/github.com/libdns/libdns
- **Caddy DNS Provider Guide**: https://caddyserver.com/docs/modules/dns.providers

## License

See LICENSE file for details.
