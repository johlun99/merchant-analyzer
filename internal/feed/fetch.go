package feed

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetch downloads the feed at url, measures size and fetch time, then parses it.
func Fetch(url string) (*Feed, error) {
	start := time.Now()

	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	elapsed := time.Since(start)

	f, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	f.URL = url
	f.Size = int64(len(data))
	f.FetchTime = elapsed
	f.Raw = data

	return f, nil
}
