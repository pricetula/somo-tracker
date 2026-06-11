package utils

import (
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"

	"go.uber.org/fx"
)

// ErrSSRFBlocked is returned when a connection to a private or loopback address is blocked.
var ErrSSRFBlocked = errors.New("ssrf: connection to private/loopback address blocked")

// privateCIDRs lists all private, loopback, link-local, and metadata IP ranges to block.
var privateCIDRs []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",   // loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // private class A
		"172.16.0.0/12",  // private class B
		"192.168.0.0/16", // private class C
		"169.254.0.0/16", // link-local / cloud metadata (169.254.169.254)
		"fe80::/10",      // IPv6 link-local
	}
	for _, c := range cidrs {
		_, cidr, err := net.ParseCIDR(c)
		if err != nil {
			panic("invalid hardcoded CIDR in ssrf blocklist: " + err.Error())
		}
		privateCIDRs = append(privateCIDRs, cidr)
	}
}

// isPrivateIP checks whether the given IP falls within any blocked CIDR range.
func isPrivateIP(ip net.IP) bool {
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// NewSafeClient creates an *http.Client that blocks connections to private
// and loopback addresses, preventing SSRF attacks including DNS rebinding.
func NewSafeClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				// Resolve the hostname to catch DNS rebinding attacks
				ips, err := net.LookupHost(host)
				if err != nil {
					return err
				}
				for _, ipStr := range ips {
					resolved := net.ParseIP(ipStr)
					if resolved != nil && isPrivateIP(resolved) {
						return ErrSSRFBlocked
					}
				}
				return nil
			}
			if isPrivateIP(ip) {
				return ErrSSRFBlocked
			}
			return nil
		},
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

// Module is an fx-compatible provider for a safe *http.Client.
var Module = fx.Provide(NewSafeClient)
