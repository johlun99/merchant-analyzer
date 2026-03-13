// Package xmlcheck validates XML structure and attribute quality.
package xmlcheck

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

const maxValueLen = 5000

// Checker validates XML structure and attribute quality.
type Checker struct{}

// NewChecker returns a new XML Checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the checker name.
func (c *Checker) Name() string {
	return "XML Validation"
}

// fieldDef pairs a required field name with its accessor function.
type fieldDef struct {
	name string
	get  func(*feed.Product) string
}

// orderedFields returns the required product fields in a deterministic order.
func orderedFields() []fieldDef {
	return []fieldDef{
		{"id", func(p *feed.Product) string { return p.ID }},
		{"title", func(p *feed.Product) string { return p.Title }},
		{"description", func(p *feed.Product) string { return p.Description }},
		{"price", func(p *feed.Product) string { return p.Price }},
		{"availability", func(p *feed.Product) string { return p.Availability }},
		{"link", func(p *feed.Product) string { return p.Link }},
		{"image_link", func(p *feed.Product) string { return p.ImageLink }},
	}
}

// Run validates the feed's raw XML for structure and attribute quality.
func (c *Checker) Run(_ context.Context, f *feed.Feed) checker.Result {
	var items []checker.Item
	status := checker.StatusOK

	// Structural check using raw bytes.
	if err := validateStructure(f.Raw); err != nil {
		items = append(items, checker.Item{
			Field:   "xml",
			Message: err.Error(),
		})
		status = checker.StatusError
	}

	// Attribute quality: one Item per field, aggregating all affected products.
	for _, fd := range orderedFields() {
		var missing, toolong []checker.AffectedProduct
		for _, p := range f.Products {
			val := fd.get(&p)
			ap := checker.AffectedProduct{ID: p.ID, Title: p.Title}
			if val == "" {
				missing = append(missing, ap)
			} else if len(val) > maxValueLen {
				toolong = append(toolong, ap)
			}
		}
		if len(missing) > 0 {
			items = append(items, checker.Item{
				Field:            fd.name,
				Message:          fmt.Sprintf("%d products missing required field %q", len(missing), fd.name),
				Count:            len(missing),
				AffectedProducts: missing,
			})
			if status < checker.StatusWarning {
				status = checker.StatusWarning
			}
		}
		if len(toolong) > 0 {
			items = append(items, checker.Item{
				Field:            fd.name,
				Message:          fmt.Sprintf("%d products with field %q exceeding %d characters", len(toolong), fd.name, maxValueLen),
				Count:            len(toolong),
				AffectedProducts: toolong,
			})
			if status < checker.StatusWarning {
				status = checker.StatusWarning
			}
		}
	}

	return checker.Result{
		Name:   c.Name(),
		Status: status,
		Items:  items,
	}
}

// validateStructure checks that the XML is well-formed and the root element closes.
func validateStructure(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("feed data is empty")
	}

	dec := xml.NewDecoder(strings.NewReader(string(data)))
	rootClosed := false
	var rootName string

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			local := strings.ToLower(t.Name.Local)
			if rootName == "" && (local == "rss" || local == "feed") {
				rootName = local
			}
		case xml.EndElement:
			local := strings.ToLower(t.Name.Local)
			if local == rootName {
				rootClosed = true
			}
		}
	}

	if !rootClosed {
		return fmt.Errorf("feed is incomplete: root element was not closed (truncated)")
	}
	return nil
}
