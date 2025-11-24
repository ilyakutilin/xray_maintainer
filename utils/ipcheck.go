package utils

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

func removeBrackets(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}

func IsValidIP(ipStr string) bool {
	ipStr = removeBrackets(ipStr)

	if ip := net.ParseIP(ipStr); ip != nil {
		return true
	}

	return false
}

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

func IsValidVlessPort(port int) bool {
	return port == 80 || port == 443
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
