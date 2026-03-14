package exporter

import (
	"bytes"
	"encoding/csv"
)

// ToProductCSV returns a CSV with one row per affected product per issue.
// Columns: checker, field, message, product_id, product_title.
//
//nolint:gosec // formula injection in CSV fields is accepted as out of scope
func ToProductCSV(report Report) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"checker", "field", "message", "product_id", "product_title"})
	for _, result := range report.Results {
		for _, item := range result.Items {
			for _, p := range item.AffectedProducts {
				_ = w.Write([]string{result.Name, item.Field, item.Message, p.ID, p.Title})
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
