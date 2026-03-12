package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johlun99/merchant-analyzer/internal/checker"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{2 * 1024 * 1024, "2.0 MB"},
	}
	for _, tc := range tests {
		got := formatBytes(tc.input)
		if got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status checker.Status
		want   string
	}{
		{checker.StatusOK, "[OK]     "},
		{checker.StatusWarning, "[WARNING]"},
		{checker.StatusError, "[ERROR]  "},
		{checker.StatusFatal, "[FATAL]  "},
	}
	for _, tc := range tests {
		got := statusLabel(tc.status)
		if got != tc.want {
			t.Errorf("statusLabel(%v) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestFetchLatestVersionReturnsLatestWhenNewer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v9.9.9"})
	}))
	defer srv.Close()

	got, ok := fetchLatestVersionFromURL(srv.URL, 2*time.Second)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "v9.9.9" {
		t.Errorf("got %q, want v9.9.9", got)
	}
}

func TestFetchLatestVersionReturnsFalseWhenSameVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": version})
	}))
	defer srv.Close()

	_, ok := fetchLatestVersionFromURL(srv.URL, 2*time.Second)
	if ok {
		t.Error("expected ok=false when tag matches current version")
	}
}

func TestFetchLatestVersionReturnsFalseOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, ok := fetchLatestVersionFromURL(srv.URL, 2*time.Second)
	if ok {
		t.Error("expected ok=false on server error")
	}
}

func TestFetchLatestVersionReturnsFalseOnTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	_, ok := fetchLatestVersionFromURL(srv.URL, 10*time.Millisecond)
	if ok {
		t.Error("expected ok=false on timeout")
	}
}
