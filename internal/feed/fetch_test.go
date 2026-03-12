package feed_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/johlun99/merchant-analyzer/internal/feed"
)

func TestFetchReturnsContent(t *testing.T) {
	data, err := os.ReadFile("../../testdata/valid_feed.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	result, err := feed.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.URL != srv.URL {
		t.Errorf("URL = %q, want %q", result.URL, srv.URL)
	}
	if result.Size != int64(len(data)) {
		t.Errorf("Size = %d, want %d", result.Size, len(data))
	}
	if result.FetchTime <= 0 {
		t.Error("FetchTime should be positive")
	}
}

func TestFetchMeasuresTime(t *testing.T) {
	delay := 10 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel></channel></rss>`))
	}))
	defer srv.Close()

	result, err := feed.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FetchTime < delay {
		t.Errorf("FetchTime %v should be >= %v", result.FetchTime, delay)
	}
}

func TestFetchErrorOnNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := feed.Fetch(srv.URL)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestFetchErrorOnInvalidURL(t *testing.T) {
	_, err := feed.Fetch("not-a-valid-url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestFromFileReturnsContent(t *testing.T) {
	path := "../../testdata/valid_feed.xml"
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}

	result, err := feed.FromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.URL != path {
		t.Errorf("URL = %q, want %q", result.URL, path)
	}
	if result.Size != info.Size() {
		t.Errorf("Size = %d, want %d", result.Size, info.Size())
	}
	if result.FetchTime != 0 {
		t.Errorf("FetchTime = %v, want 0 for file input", result.FetchTime)
	}
	if result.ProductCount == 0 {
		t.Error("ProductCount should be > 0 for valid feed")
	}
}

func TestFromFileErrorOnMissingFile(t *testing.T) {
	_, err := feed.FromFile("/nonexistent/path/feed.xml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
