// Package adstxt provides functionality for fetching and parsing ads.txt files.
// It implements the IAB ads.txt specification for extracting advertiser domains.
package adstxt

import (
	"regexp"
	"strings"
)

// AdvertiserCount represents an advertiser domain and the number of times it appears in an ads.txt file.
type AdvertiserCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// linePattern matches valid ads.txt lines that start with a domain name.
// Format: domain.com,publisher_id,relationship,certification_authority_id
var linePattern = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z0-9][a-zA-Z0-9-]*),`)

// ParseAdsTxt parses the content of an ads.txt file and returns a map of advertiser domains to their counts.
// It ignores empty lines and comments (lines starting with #).
// Domain names are normalized to lowercase for case-insensitive counting.
func ParseAdsTxt(content string) map[string]int {
	advertisers := make(map[string]int)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := linePattern.FindStringSubmatch(line)
		if len(matches) >= 2 {
			domain := strings.ToLower(matches[1])
			advertisers[domain]++
		}
	}

	return advertisers
}

// MapToSlice converts a map of advertiser domains and counts to a slice of AdvertiserCount structs.
// This is useful for JSON serialization where the order can be controlled by sorting.
func MapToSlice(advertisers map[string]int) []AdvertiserCount {
	result := make([]AdvertiserCount, 0, len(advertisers))
	for domain, count := range advertisers {
		result = append(result, AdvertiserCount{
			Domain: domain,
			Count:  count,
		})
	}
	return result
}
