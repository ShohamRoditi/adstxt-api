package adstxt

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const maxResponseSize = 10 << 20 // 10MB max size for ads.txt files

// Fetcher handles HTTP requests to retrieve ads.txt files from domains.
// It tries multiple URL patterns (https, http, www prefix) to maximize success.
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// NewFetcher creates a new Fetcher with the specified timeout.
// The Fetcher limits redirects to 10 to prevent infinite redirect loops.
// It also sets granular timeouts for connection, TLS handshake, and response headers.
func NewFetcher(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,  // Connection timeout
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   5 * time.Second, // TLS handshake timeout
				ResponseHeaderTimeout: 5 * time.Second, // Response header timeout
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		timeout: timeout,
	}
}

// FetchAdsTxt retrieves the ads.txt file content for the given domain.
// It attempts to fetch from multiple URL patterns in order:
//  1. https://domain/ads.txt
//  2. http://domain/ads.txt
//  3. https://www.domain/ads.txt
//
// Returns the content of the first successful response, or an error if all attempts fail.
func (f *Fetcher) FetchAdsTxt(domain string) (string, error) {
	urls := []string{
		fmt.Sprintf("https://%s/ads.txt", domain),
		fmt.Sprintf("http://%s/ads.txt", domain),
		fmt.Sprintf("https://www.%s/ads.txt", domain),
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	var lastErr error
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("User-Agent", "AdsTxtBot/1.0")

		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Limit response size to prevent DoS attacks
			limitedReader := io.LimitReader(resp.Body, maxResponseSize)
			body, err := io.ReadAll(limitedReader)
			if err != nil {
				lastErr = err
				continue
			}
			return string(body), nil
		}
		lastErr = fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return "", fmt.Errorf("failed to fetch ads.txt for %s: %v", domain, lastErr)
}
