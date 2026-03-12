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

func reportWithExamples() exporter.Report {
	return exporter.Report{
		URL:          "https://example.com/feed.xml",
		FetchTime:    500 * time.Millisecond,
		Size:         1024,
		ProductCount: 10,
		Results: []checker.Result{
			{
				Name:   "Google Feed Spec",
				Status: checker.StatusError,
				Items: []checker.Item{
					{
						Field:    "price",
						Message:  "2 of 10 products missing required field \"price\"",
						Count:    2,
						Examples: []string{`prod-001 "Blue Shoes" — price: (missing)`, `prod-002 "Red Shoes" — price: (missing)`},
					},
				},
			},
		},
	}
}

func TestMarkdownExportContainsExamplesSection(t *testing.T) {
	md := exporter.ToMarkdown(reportWithExamples())
	if !strings.Contains(md, "## Examples") {
		t.Error("Markdown missing \"## Examples\" section")
	}
}

func TestMarkdownExportExamplesSectionContainsCheckerName(t *testing.T) {
	md := exporter.ToMarkdown(reportWithExamples())
	if !strings.Contains(md, "Google Feed Spec") {
		t.Error("Examples section should contain checker name")
	}
}

func TestMarkdownExportExamplesSectionContainsFieldHeader(t *testing.T) {
	md := exporter.ToMarkdown(reportWithExamples())
	if !strings.Contains(md, "**price**") {
		t.Error("Examples section should contain bold field header")
	}
}

func TestMarkdownExportExamplesSectionContainsBullets(t *testing.T) {
	md := exporter.ToMarkdown(reportWithExamples())
	if !strings.Contains(md, `- prod-001 "Blue Shoes" — price: (missing)`) {
		t.Error("Examples section should contain example bullets")
	}
}

func TestMarkdownExportNoExamplesSectionWhenNone(t *testing.T) {
	md := exporter.ToMarkdown(sampleReport())
	if strings.Contains(md, "## Examples") {
		t.Error("Markdown should not contain Examples section when no examples exist")
	}
}

func TestJSONExportContainsExamples(t *testing.T) {
	data, err := exporter.ToJSON(reportWithExamples())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"examples"`) {
		t.Error("JSON missing \"examples\" field")
	}
	if !strings.Contains(body, "Blue Shoes") {
		t.Error("JSON examples missing expected content")
	}
}

func reportWithAttributes() exporter.Report {
	return exporter.Report{
		URL:          "https://example.com/feed.xml",
		FetchTime:    500 * time.Millisecond,
		Size:         1024,
		ProductCount: 10,
		Attributes: []exporter.AttributeGroup{
			{
				Category: "Required",
				Items: []exporter.Attribute{
					{Name: "id", Tags: []string{"Required", "AI"}, Coverage: 100},
					{Name: "title", Tags: []string{"Required", "AI"}, Coverage: 100},
				},
			},
			{
				Category: "Recommended",
				Items: []exporter.Attribute{
					{Name: "brand", Tags: []string{"Recommended", "AI"}, Coverage: 80},
				},
			},
			{Category: "AI", Items: nil},
			{Category: "Supported", Items: nil},
			{
				Category: "Custom",
				Items: []exporter.Attribute{
					{Name: "custom_field", Tags: []string{"Custom"}, Coverage: 40},
				},
			},
		},
	}
}

func TestAttributeGroupsInJSON(t *testing.T) {
	data, err := exporter.ToJSON(reportWithAttributes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"attributes"`) {
		t.Error(`JSON missing "attributes" key`)
	}
	if !strings.Contains(body, `"Required"`) {
		t.Error(`JSON attributes missing "Required" category`)
	}
	if !strings.Contains(body, `"coverage"`) {
		t.Error(`JSON attributes missing "coverage" field`)
	}
}

func TestAttributeGroupsEmptyCategoryOmittedJSON(t *testing.T) {
	data, err := exporter.ToJSON(reportWithAttributes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(data)
	// "AI" and "Supported" groups have no items and should be omitted
	// We check that only non-empty categories appear as top-level entries
	// by verifying the total count (Required, Recommended, Custom = 3)
	_ = body // structural check via JSON unmarshal
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	attrs, ok := out["attributes"].([]any)
	if !ok {
		t.Fatal("attributes is not an array")
	}
	for _, g := range attrs {
		grp := g.(map[string]any)
		items, _ := grp["items"].([]any)
		if len(items) == 0 {
			t.Errorf("empty group %q should be omitted from JSON", grp["category"])
		}
	}
}

func TestAttributeGroupsInMarkdown(t *testing.T) {
	md := exporter.ToMarkdown(reportWithAttributes())
	for _, want := range []string{"## Attributes", "### Required", "### Recommended", "### Custom", "Coverage"} {
		if !strings.Contains(md, want) {
			t.Errorf("Markdown missing %q", want)
		}
	}
}

func TestAttributeGroupsEmptyCategoryOmittedMarkdown(t *testing.T) {
	md := exporter.ToMarkdown(reportWithAttributes())
	// "AI" and "Supported" have no items — their headers should not appear
	if strings.Contains(md, "### AI\n") {
		t.Error("Markdown should omit empty AI group")
	}
	if strings.Contains(md, "### Supported\n") {
		t.Error("Markdown should omit empty Supported group")
	}
}

func TestJSONExportNoExamplesKeyWhenNone(t *testing.T) {
	data, err := exporter.ToJSON(sampleReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(data), `"examples"`) {
		t.Error("JSON should omit \"examples\" key when empty")
	}
}
