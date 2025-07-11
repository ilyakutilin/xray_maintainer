package utils

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

func IsValidIPOrCIDR(ipStr string) bool {
	// Try parsing as CIDR (IPv4 or IPv6 with range)
	if _, _, err := net.ParseCIDR(ipStr); err == nil {
		return true
	}

	// Try parsing as a plain IP (IPv4 or IPv6)
	if ip := net.ParseIP(ipStr); ip != nil {
		return true
	}

	return false
}

func IsExternalIPv4(ipStr string) bool {
	// Parse the IP address
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		return false // Not a valid IPv4
	}

	// Check for special cases
	if ipStr == "0.0.0.0" || ipStr == "localhost" {
		return false
	}

	// Check for private/local IP ranges
	if ip.IsPrivate() {
		return false
	}

	ip = ip.To4()

	// Check for other special ranges
	if isLinkLocal(ip) || isLoopback(ip) || isMulticast(ip) || isReserved(ip) {
		return false
	}

	return true
}

// Helper functions to check specific IP ranges
func isLinkLocal(ip net.IP) bool {
	// 169.254.0.0/16
	return ip[0] == 169 && ip[1] == 254
}

func isLoopback(ip net.IP) bool {
	// 127.0.0.0/8
	return ip[0] == 127
}

func isMulticast(ip net.IP) bool {
	// 224.0.0.0/4
	return ip[0] >= 224 && ip[0] <= 239
}

func isReserved(ip net.IP) bool {
	// Check various reserved ranges
	switch {
	case ip[0] == 0: // 0.0.0.0/8 (already checked 0.0.0.0 specifically)
		return true
	case ip[0] == 192 && ip[1] == 0 && ip[2] == 0: // 192.0.0.0/24 (IANA)
		return true
	// case ip[0] == 192 && ip[1] == 0 && ip[2] == 2: // 192.0.2.0/24 (TEST-NET-1)
	// 	return true
	case ip[0] == 192 && ip[1] == 88 && ip[2] == 99: // 192.88.99.0/24 (6to4 relay anycast)
		return true
	case ip[0] == 198 && ip[1] == 18: // 198.18.0.0/15 (benchmarking)
		return true
	// case ip[0] == 198 && ip[1] == 51 && ip[2] == 100: // 198.51.100.0/24 (TEST-NET-2)
	// 	return true
	// case ip[0] == 203 && ip[1] == 0 && ip[2] == 113: // 203.0.113.0/24 (TEST-NET-3)
	// 	return true
	case ip[0] >= 240: // 240.0.0.0/4 (reserved)
		return true
	}
	return false
}

func GetCountryCode(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := GetRequestWithProxy(
		ctx, "http://ip-api.com/line/?fields=countryCode", nil)
	if err != nil {
		return "", fmt.Errorf("failed to obtain the country information: %w", err)
	}
	countryCode := strings.Replace(string(resp), "\n", "", -1)

	isValidCountryCode, _ := regexp.MatchString(`^[A-Z]{2}$`, string(countryCode))
	if !isValidCountryCode {
		return "", fmt.Errorf("%s is not a valid ISO 3166-1 alpha-2 country code",
			countryCode)
	}

	return string(countryCode), nil
}
