package adstxt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

func NewFetcher(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
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

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = err
				continue
			}
			return string(body), nil
		}
		resp.Body.Close()
		lastErr = fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return "", fmt.Errorf("failed to fetch ads.txt for %s: %v", domain, lastErr)
}
