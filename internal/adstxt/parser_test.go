package adstxt

import (
	"testing"
)

func TestParseAdsTxt_ValidContent(t *testing.T) {
	content := `google.com, pub-1234567890, DIRECT, f08c47fec0942fa0
appnexus.com, 12345, RESELLER, f5ab79cb980f11d1
google.com, pub-9876543210, DIRECT, f08c47fec0942fa0
# This is a comment
rubiconproject.com, 23456, RESELLER, 0bfd66d529a55807`

	advertisers := ParseAdsTxt(content)

	if len(advertisers) != 3 {
		t.Errorf("Expected 3 unique advertisers, got %d", len(advertisers))
	}

	if advertisers["google.com"] != 2 {
		t.Errorf("Expected google.com count 2, got %d", advertisers["google.com"])
	}

	if advertisers["appnexus.com"] != 1 {
		t.Errorf("Expected appnexus.com count 1, got %d", advertisers["appnexus.com"])
	}
}

func TestParseAdsTxt_EmptyContent(t *testing.T) {
	content := ""
	advertisers := ParseAdsTxt(content)

	if len(advertisers) != 0 {
		t.Errorf("Expected 0 advertisers for empty content, got %d", len(advertisers))
	}
}

func TestParseAdsTxt_OnlyComments(t *testing.T) {
	content := `# Comment 1
# Comment 2
# Comment 3`

	advertisers := ParseAdsTxt(content)

	if len(advertisers) != 0 {
		t.Errorf("Expected 0 advertisers for comment-only content, got %d", len(advertisers))
	}
}

func TestParseAdsTxt_InvalidLines(t *testing.T) {
	content := `google.com, pub-1234567890, DIRECT
invalid line without comma
appnexus.com, 12345, RESELLER
another invalid line`

	advertisers := ParseAdsTxt(content)

	if len(advertisers) != 2 {
		t.Errorf("Expected 2 valid advertisers, got %d", len(advertisers))
	}
}

func TestMapToSlice(t *testing.T) {
	advertisers := map[string]int{
		"google.com":   5,
		"appnexus.com": 3,
		"rubicon.com":  1,
	}

	slice := MapToSlice(advertisers)

	if len(slice) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(slice))
	}

	found := make(map[string]int)
	for _, adv := range slice {
		found[adv.Domain] = adv.Count
	}

	for domain, count := range advertisers {
		if found[domain] != count {
			t.Errorf("Expected %s count %d, got %d", domain, count, found[domain])
		}
	}
}
