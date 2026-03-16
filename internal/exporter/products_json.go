package exporter

import (
	"encoding/json"

	"github.com/johlun99/merchant-analyzer/internal/checker"
)

// ProductIssue is a single checker finding with its full affected product list.
type ProductIssue struct {
	Checker          string                    `json:"checker"`
	Field            string                    `json:"field"`
	Message          string                    `json:"message"`
	Impact           string                    `json:"impact,omitempty"`
	ImpactDesc       string                    `json:"impact_desc,omitempty"`
	AffectedProducts []checker.AffectedProduct `json:"affected_products"`
}

// ToProductJSON returns a JSON array of issues with their full affected product lists.
// Items with no affected products are omitted.
func ToProductJSON(report Report) ([]byte, error) {
	var issues []ProductIssue
	for _, result := range report.Results {
		for _, item := range result.Items {
			if len(item.AffectedProducts) == 0 {
				continue
			}
			issues = append(issues, ProductIssue{
				Checker:          result.Name,
				Field:            item.Field,
				Message:          item.Message,
				Impact:           item.Impact,
				ImpactDesc:       item.ImpactDesc,
				AffectedProducts: item.AffectedProducts,
			})
		}
	}
	return json.MarshalIndent(issues, "", "  ")
}
