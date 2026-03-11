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

	// Attribute quality: check for empty required fields and excessively long values.
	for i, p := range f.Products {
		fields := productFields(&p)
		for field, value := range fields {
			if value == "" {
				items = append(items, checker.Item{
					Field:   field,
					Message: fmt.Sprintf("product %d: required field %q is empty", i+1, field),
					Count:   1,
				})
				if status < checker.StatusWarning {
					status = checker.StatusWarning
				}
				continue
			}
			if len(value) > maxValueLen {
				items = append(items, checker.Item{
					Field:   field,
					Message: fmt.Sprintf("product %d: field %q exceeds %d characters (%d)", i+1, field, maxValueLen, len(value)),
					Count:   1,
				})
				if status < checker.StatusWarning {
					status = checker.StatusWarning
				}
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

// productFields returns a map of required field name → value for a product.
func productFields(p *feed.Product) map[string]string {
	return map[string]string{
		"id":           p.ID,
		"title":        p.Title,
		"description":  p.Description,
		"price":        p.Price,
		"availability": p.Availability,
		"link":         p.Link,
		"image_link":   p.ImageLink,
	}
}
