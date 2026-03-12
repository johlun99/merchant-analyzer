package views

import (
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
)

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
