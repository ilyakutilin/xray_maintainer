package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type CFCreds struct {
	SecretKey string
	PublicKey string
	Reserved  []int
	V4        string
	V6        string
	Endpoint  string
}

// Parses the Cloudflare generator output. Tailored specifically for the output of
// github.com/badafans/warp-reg.
func ParseCFCreds(output string) (CFCreds, error) {
	var result CFCreds

	patterns := map[string]*regexp.Regexp{
		"private_key": regexp.MustCompile(`(?m)^private_key:\s*(\S+)`),
		"public_key":  regexp.MustCompile(`(?m)^public_key:\s*(\S+)`),
		"reserved":    regexp.MustCompile(`(?m)^reserved:\s*\[([0-9,\s]+)\]`),
		"v4":          regexp.MustCompile(`(?m)^v4:\s*(\S+)`),
		"v6":          regexp.MustCompile(`(?m)^v6:\s*(\S+)`),
		"endpoint":    regexp.MustCompile(`(?m)^endpoint:\s*(\S+)`),
	}

	for key, pattern := range patterns {
		matches := pattern.FindStringSubmatch(output)
		if len(matches) < 2 {
			return result, errors.New("missing required field: " + key)
		}
		switch key {
		case "private_key":
			result.SecretKey = matches[1]
		case "public_key":
			result.PublicKey = matches[1]
		case "reserved":
			values := strings.Split(matches[1], ",")
			for _, v := range values {
				var num int
				fmt.Sscanf(strings.TrimSpace(v), "%d", &num)
				result.Reserved = append(result.Reserved, num)
			}
		case "v4":
			result.V4 = matches[1]
		case "v6":
			result.V6 = matches[1]
		case "endpoint":
			result.Endpoint = matches[1]
		}
	}

	return result, nil
}

func updateWarp(xrayServerIP string, xrayProtocol string, xrayClientPort int, ipCheckerURL string, cfCredGenURL string) error {
	// TODO: Implement the updateCFCreds function
	return nil
}
