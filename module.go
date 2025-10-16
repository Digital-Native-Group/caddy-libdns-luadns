package luadns

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/libdns/luadns"
)

// Provider lets Caddy read and manipulate DNS records hosted by Lua DNS.
type Provider struct{ *luadns.Provider }

func init() {
	caddy.RegisterModule(Provider{})
}

// CaddyModule returns the Caddy module information.
func (Provider) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "dns.providers.luadns",
		New: func() caddy.Module { return &Provider{new(luadns.Provider)} },
	}
}

// Provision sets up the module. Implements caddy.Provisioner.
func (p *Provider) Provision(ctx caddy.Context) error {
	repl := caddy.NewReplacer()
	p.Provider.Email = repl.ReplaceAll(p.Provider.Email, "")
	p.Provider.APIKey = repl.ReplaceAll(p.Provider.APIKey, "")
	return nil
}

// UnmarshalCaddyfile sets up the DNS provider from Caddyfile tokens. Syntax:
//
//	luadns {
//	    email <email>
//	    api_key <api_key>
//	}
//
// or inline:
//
//	luadns <email> <api_key>
func (p *Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		// Support inline format: luadns <email> <api_key>
		args := d.RemainingArgs()
		if len(args) == 2 {
			p.Provider.Email = args[0]
			p.Provider.APIKey = args[1]
		} else if len(args) > 0 {
			return d.ArgErr()
		}

		// Parse block format
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "email":
				if p.Provider.Email != "" {
					return d.Err("email already set")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.Provider.Email = d.Val()
				if d.NextArg() {
					return d.ArgErr()
				}
			case "api_key":
				if p.Provider.APIKey != "" {
					return d.Err("API key already set")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.Provider.APIKey = d.Val()
				if d.NextArg() {
					return d.ArgErr()
				}
			default:
				return d.Errf("unrecognized subdirective '%s'", d.Val())
			}
		}
	}

	// Validate required fields
	if p.Provider.Email == "" {
		return d.Err("missing email")
	}
	if p.Provider.APIKey == "" {
		return d.Err("missing API key")
	}

	return nil
}

// Interface guards
var (
	_ caddyfile.Unmarshaler = (*Provider)(nil)
	_ caddy.Provisioner     = (*Provider)(nil)
)
