package dns

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"
)

var providers = []string{
	"8.8.8.8:53", // Google
	"8.8.4.4:53", // Google
	"1.1.1.1:53", // Cloudflare
	"1.0.0.1:53", // Cloudflare
}

type Resolver struct {
	internal *net.Resolver
}

func NewResolver() *Resolver {
	return &Resolver{
		internal: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 2 * time.Second,
				}
				// Pick a random public provider
				// Note: In Go 1.20+, global rand is seeded automatically.
				provider := providers[rand.Intn(len(providers))]
				return d.DialContext(ctx, "udp", provider)
			},
		},
	}
}

// DialContext resolves the hostname using custom DNS and then dials the IP.
func (r *Resolver) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// 1. Resolve IP (using our custom resolver)
	ips, err := r.internal.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}

	// 2. Dial IP (try all resolved IPs)
	for _, ip := range ips {
		d := net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		target := net.JoinHostPort(ip, port)
		conn, err := d.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("failed to reach %s", host)
}
