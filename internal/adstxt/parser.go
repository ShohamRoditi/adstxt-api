package adstxt

import (
	"regexp"
	"strings"
)

type AdvertiserCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

var linePattern = regexp.MustCompile(`^([a-zA-Z0-9.-]+\.[a-zA-Z]{2,}),`)

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
