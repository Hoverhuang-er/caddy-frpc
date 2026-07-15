package frpc

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
// Syntax:
//
//	frpc <config_path>
//
//	frpc {
//	    config <config_path>
//	}
//
// Examples:
//
//	frpc ./frpc.toml
//
//	frpc {
//	    config ./frpc.yaml
//	}
func (f *FRPC) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			// Single argument: config file path
			f.ConfigFile = d.Val()
			if d.NextArg() {
				return d.ArgErr()
			}
			continue
		}
		// Block mode with subdirectives
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "config":
				if !d.AllArgs(&f.ConfigFile) {
					return d.ArgErr()
				}
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}
	return nil
}

// Interface assertions
var (
	_ caddyfile.Unmarshaler = (*FRPC)(nil)
)
