package workers

import (
	"fmt"
	"net"
	"net/url"
)

// IsSafeURL parses the URL and ensures it doesn't resolve to a private/local IP address (SSRF prevention).
func IsSafeURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	hostname := u.Hostname()
	if hostname == "" {
		return false
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		fmt.Printf("DNS Lookup failed for %s: %v\n", hostname, err)
		return false
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
			fmt.Printf("SSRF Blocked: URL resolved to private/local IP (%s -> %s)\n", hostname, ip.String())
			return false
		}
	}

	return true
}
