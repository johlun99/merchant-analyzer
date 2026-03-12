package exporter_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/exporter"
)

func sampleReport() exporter.Report {
	score := 74
	return exporter.Report{
		URL:          "https://example.com/feed.xml",
		FetchTime:    1200 * time.Millisecond,
		Size:         2400000,
		ProductCount: 1203,
		Results: []checker.Result{
			{
				Name:   "XML Validation",
				Status: checker.StatusOK,
				Items:  nil,
			},
			{
				Name:   "Attribute Check",
				Status: checker.StatusWarning,
				Items: []checker.Item{
					{Field: "description", Message: "3.2% of products missing description", Count: 38},
				},
			},
			{
				Name:   "AI Readiness",
				Status: checker.StatusWarning,
				Score:  &score,
				Items:  []checker.Item{},
			},
		},
	}
}

func TestJSONExportIsValidJSON(t *testing.T) {
	report := sampleReport()
	data, err := exporter.ToJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestJSONExportContainsKeyFields(t *testing.T) {
	report := sampleReport()
	data, err := exporter.ToJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := string(data)
	for _, want := range []string{"url", "fetch_time_ms", "size_bytes", "product_count", "results"} {
		if !strings.Contains(body, want) {
			t.Errorf("JSON missing field %q", want)
		}
	}
}

func TestMarkdownExportContainsSections(t *testing.T) {
	report := sampleReport()
	md := exporter.ToMarkdown(report)

	for _, want := range []string{
		"# merchant-analyzer",
		"example.com/feed.xml",
		"XML Validation",
		"Attribute Check",
		"AI Readiness",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("Markdown missing %q", want)
		}
	}
}

func TestMarkdownExportContainsScore(t *testing.T) {
	report := sampleReport()
	md := exporter.ToMarkdown(report)

	if !strings.Contains(md, "74") {
		t.Error("Markdown should contain the AI readiness score (74)")
	}
}

func TestMarkdownExportContainsMetrics(t *testing.T) {
	report := sampleReport()
	md := exporter.ToMarkdown(report)

	if !strings.Contains(md, "1203") {
		t.Error("Markdown should contain product count")
	}
}
