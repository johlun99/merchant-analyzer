package exporter_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/exporter"
)

func productReport() exporter.Report {
	return exporter.Report{
		ProductCount: 3,
		Results: []checker.Result{
			{
				Name:   "XML Validation",
				Status: checker.StatusWarning,
				Items: []checker.Item{
					{
						Field:   "title",
						Message: "2 products missing required field \"title\"",
						Count:   2,
						AffectedProducts: []checker.AffectedProduct{
							{ID: "sku-001", Title: ""},
							{ID: "sku-002", Title: ""},
						},
					},
				},
			},
			{
				Name:   "Google Feed Spec",
				Status: checker.StatusWarning,
				Items: []checker.Item{
					{
						Field:   "brand",
						Message: "1 of 3 products missing recommended field \"brand\"",
						Count:   1,
						AffectedProducts: []checker.AffectedProduct{
							{ID: "sku-003", Title: "Widget"},
						},
					},
					// Item with no affected products — should be omitted from JSON output.
					{
						Field:   "gtin",
						Message: "0 violations",
						Count:   0,
					},
				},
			},
		},
	}
}

// TestToProductCSVHeaders verifies the CSV header row.
func TestToProductCSVHeaders(t *testing.T) {
	data, err := exporter.ToProductCSV(productReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	csv := string(data)
	want := "checker,field,message,product_id,product_title"
	if !strings.HasPrefix(csv, want) {
		t.Errorf("CSV does not start with expected header; got:\n%s", csv)
	}
}

// TestToProductCSVRowCount verifies one row per affected product.
func TestToProductCSVRowCount(t *testing.T) {
	data, err := exporter.ToProductCSV(productReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// 1 header + 2 XML items + 1 GoogleSpec brand item = 4
	const want = 4
	if len(lines) != want {
		t.Errorf("CSV line count = %d, want %d\n%s", len(lines), want, string(data))
	}
}

// TestToProductCSVFields verifies a data row contains expected values.
func TestToProductCSVFields(t *testing.T) {
	data, err := exporter.ToProductCSV(productReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	csv := string(data)
	if !strings.Contains(csv, "Google Feed Spec") {
		t.Error("CSV should contain checker name 'Google Feed Spec'")
	}
	if !strings.Contains(csv, "sku-003") {
		t.Error("CSV should contain product ID 'sku-003'")
	}
	if !strings.Contains(csv, "Widget") {
		t.Error("CSV should contain product title 'Widget'")
	}
}

// TestToProductCSVEmptyReport verifies a header-only output for a report with no affected products.
func TestToProductCSVEmptyReport(t *testing.T) {
	empty := exporter.Report{
		Results: []checker.Result{
			{Name: "XML Validation", Items: []checker.Item{
				{Field: "xml", Message: "ok", Count: 0},
			}},
		},
	}
	data, err := exporter.ToProductCSV(empty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected header only (1 line), got %d lines", len(lines))
	}
}

// TestToProductJSONStructure verifies the top-level JSON array structure.
func TestToProductJSONStructure(t *testing.T) {
	data, err := exporter.ToProductJSON(productReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var issues []exporter.ProductIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, string(data))
	}
	// gtin item has no AffectedProducts → should be omitted; expect 2 issues
	if len(issues) != 2 {
		t.Errorf("issues count = %d, want 2", len(issues))
	}
}

// TestToProductJSONOmitsZeroProductItems verifies items with no affected products are skipped.
func TestToProductJSONOmitsZeroProductItems(t *testing.T) {
	data, err := exporter.ToProductJSON(productReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(data), "gtin") {
		t.Error("JSON should not contain item 'gtin' which has no affected products")
	}
}

// TestToProductJSONAffectedProducts verifies products are correctly nested.
func TestToProductJSONAffectedProducts(t *testing.T) {
	data, err := exporter.ToProductJSON(productReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var issues []exporter.ProductIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, issue := range issues {
		if issue.Field == "brand" {
			if len(issue.AffectedProducts) != 1 {
				t.Fatalf("brand AffectedProducts len = %d, want 1", len(issue.AffectedProducts))
			}
			if issue.AffectedProducts[0].ID != "sku-003" {
				t.Errorf("product ID = %q, want sku-003", issue.AffectedProducts[0].ID)
			}
			return
		}
	}
	t.Error("no 'brand' issue found in JSON output")
}

// TestToProductJSONEmptyReport verifies an empty JSON array is returned when nothing is affected.
func TestToProductJSONEmptyReport(t *testing.T) {
	empty := exporter.Report{
		Results: []checker.Result{
			{Name: "XML Validation", Items: []checker.Item{
				{Field: "xml", Message: "ok", Count: 0},
			}},
		},
	}
	data, err := exporter.ToProductJSON(empty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var issues []exporter.ProductIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}
