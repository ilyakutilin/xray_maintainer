package utils

import (
	"net"
	"regexp"
	"strconv"
)

// IsValidEndpoint checks if the input string is in either URL:port or IP:port format
func IsValidEndpoint(endpoint string) bool {
	// Split host and port
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}

	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return false
	}

	// Check for IPv6 (already unwrapped from brackets by SplitHostPort)
	if ip := net.ParseIP(host); ip != nil {
		return true
	}

	// Validate domain or localhost
	domainRegex := regexp.MustCompile(`^(localhost|([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,})$`)

	return domainRegex.MatchString(host)
}
