package service

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

var blockedNetworks = parsePrefixes([]string{
	// IPv4: частные, локальные и специальные диапазоны.
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",

	// IPv6: локальные, специальные и зарезервированные диапазоны.
	"::/128",
	"::1/128",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/23",
	"2001:db8::/32",
	"2002::/16",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
})

func parsePrefixes(values []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))

	for _, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			panic(fmt.Sprintf("invalid IP prefix %q: %v", value, err))
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes
}

func isPublicIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}

	addr = addr.Unmap()

	if !addr.IsValid() || !addr.IsGlobalUnicast() {
		return false
	}

	for _, prefix := range blockedNetworks {
		if prefix.Contains(addr) {
			return false
		}
	}

	return true
}

// safeDialContext проверяет DNS-ответы и подключается непосредственно
// к проверенному IP, чтобы DNS не мог подмениться между проверкой и соединением.
func safeDialContext(
	ctx context.Context,
	network string,
	address string,
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid target address: %w", err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || (portNumber != 80 && portNumber != 443) {
		return nil, fmt.Errorf("unsupported target port")
	}

	host = strings.Trim(host, "[]")

	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return nil, fmt.Errorf("connection to non-public IP is blocked")
		}

		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(
			ctx,
			network,
			net.JoinHostPort(ip.String(), port),
		)
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve target host: %w", err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("target host has no IP addresses")
	}

	// Если хотя бы один ответ DNS небезопасен, отклоняем весь домен.
	for _, candidate := range ips {
		if !isPublicIP(candidate.IP) {
			return nil, fmt.Errorf("host resolves to a non-public IP")
		}
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	var lastErr error

	for _, candidate := range ips {
		conn, err := dialer.DialContext(
			ctx,
			network,
			net.JoinHostPort(candidate.IP.String(), port),
		)
		if err == nil {
			return conn, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("connect to resolved host: %w", lastErr)
}
