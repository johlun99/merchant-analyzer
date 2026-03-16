package views

import (
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

func makeReportViewWithResults(results []checker.Result) *ReportView {
	f := &feed.Feed{URL: "http://example.com"}
	rv := NewReportView(f, results, 80, 24)
	return &rv
}

func TestOpenExportPopulatesSections(t *testing.T) {
	results := []checker.Result{
		{Name: "XML Validation"},
		{Name: "Attribute Check"},
	}
	v := makeReportViewWithResults(results)
	v.OpenExport()

	// Results + synthetic "Attributes"
	if len(v.exportSections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(v.exportSections))
	}
	if v.exportSections[0].name != "XML Validation" || !v.exportSections[0].enabled {
		t.Error("first section should be XML Validation, enabled")
	}
	if v.exportSections[2].name != "Attributes" || !v.exportSections[2].enabled {
		t.Error("last section should be Attributes, enabled")
	}
}

func TestExportSelectionsIncludedSections(t *testing.T) {
	results := []checker.Result{{Name: "XML Validation"}}
	v := makeReportViewWithResults(results)
	v.OpenExport()

	sel := v.ExportSelections()
	if sel.IncludedSections == nil {
		t.Fatal("IncludedSections should not be nil")
	}
	if !sel.IncludedSections["XML Validation"] {
		t.Error("XML Validation should be included by default")
	}
	if !sel.IncludedSections["Attributes"] {
		t.Error("Attributes should be included by default")
	}
}

func TestHandleExportToggleKeySectionToggle(t *testing.T) {
	results := []checker.Result{{Name: "XML Validation"}, {Name: "Attribute Check"}}
	v := makeReportViewWithResults(results)
	v.OpenExport()

	// Navigate to row 4 (first section = XML Validation)
	for range 4 {
		v.HandleExportToggleKey("down")
	}
	if v.exportFocusRow != 4 {
		t.Fatalf("expected focus row 4, got %d", v.exportFocusRow)
	}

	// Toggle off
	v.HandleExportToggleKey(" ")
	if v.exportSections[0].enabled {
		t.Error("XML Validation should be disabled after toggle")
	}

	// Toggle back on
	v.HandleExportToggleKey(" ")
	if !v.exportSections[0].enabled {
		t.Error("XML Validation should be re-enabled after second toggle")
	}
}

func TestHandleExportToggleKeyAllSectionsDisabledError(t *testing.T) {
	results := []checker.Result{{Name: "XML Validation"}}
	v := makeReportViewWithResults(results)
	v.OpenExport()

	// Disable XML Validation (row 4)
	for range 4 {
		v.HandleExportToggleKey("down")
	}
	v.HandleExportToggleKey(" ")

	// Disable Attributes (row 5)
	v.HandleExportToggleKey("down")
	v.HandleExportToggleKey(" ")

	// Main report is still on, all sections off → should error
	confirmed := v.HandleExportToggleKey("enter")
	if confirmed {
		t.Error("should not confirm export when all sections are disabled")
	}
	if v.exportToggleErr == "" {
		t.Error("expected error message when all sections disabled")
	}
}

func TestHandleExportToggleKeyNoSectionErrorWhenMainOff(t *testing.T) {
	results := []checker.Result{{Name: "XML Validation"}}
	v := makeReportViewWithResults(results)
	v.OpenExport()

	// Turn off main report (row 1)
	v.HandleExportToggleKey("down")
	v.HandleExportToggleKey(" ") // exportMain = false

	// Disable all sections
	for range 3 {
		v.HandleExportToggleKey("down")
	}
	v.HandleExportToggleKey(" ") // disable XML Validation
	v.HandleExportToggleKey("down")
	v.HandleExportToggleKey(" ") // disable Attributes

	// CSV is still on, so export should proceed (no section check needed)
	confirmed := v.HandleExportToggleKey("enter")
	if !confirmed {
		t.Error("should confirm export when main is off even if all sections are disabled")
	}
}

func TestHandleExportToggleKeyTotalRowsIncludesSections(t *testing.T) {
	results := []checker.Result{{Name: "XML Validation"}, {Name: "Attribute Check"}}
	v := makeReportViewWithResults(results)
	v.OpenExport()

	// totalRows = 4 + 3 (2 results + Attributes) = 7
	// Cycling through all rows should wrap back to 0
	for range 7 {
		v.HandleExportToggleKey("down")
	}
	if v.exportFocusRow != 0 {
		t.Errorf("expected wrap to row 0, got %d", v.exportFocusRow)
	}
}

func TestImpactRankOrder(t *testing.T) {
	if impactRank("High") >= impactRank("Medium") {
		t.Error("High should rank lower (higher priority) than Medium")
	}
	if impactRank("Medium") >= impactRank("Low") {
		t.Error("Medium should rank lower (higher priority) than Low")
	}
	if impactRank("") != impactRank("Low") {
		t.Error("empty impact should rank same as Low")
	}
}

func TestSortedByImpactOrder(t *testing.T) {
	items := []checker.Item{
		{Field: "a", Impact: "Low"},
		{Field: "b", Impact: "High"},
		{Field: "c", Impact: "Medium"},
		{Field: "d", Impact: "High"},
	}
	sorted := sortedByImpact(items)

	want := []string{"High", "High", "Medium", "Low"}
	for i, item := range sorted {
		if item.Impact != want[i] {
			t.Errorf("position %d: got %q, want %q", i, item.Impact, want[i])
		}
	}
}

func TestSortedByImpactDoesNotMutateOriginal(t *testing.T) {
	items := []checker.Item{
		{Field: "a", Impact: "Low"},
		{Field: "b", Impact: "High"},
	}
	_ = sortedByImpact(items)
	if items[0].Impact != "Low" {
		t.Error("sortedByImpact must not mutate the original slice")
	}
}

func TestSortedByImpactEmpty(t *testing.T) {
	sorted := sortedByImpact(nil)
	if len(sorted) != 0 {
		t.Errorf("expected empty result, got %d items", len(sorted))
	}
}
