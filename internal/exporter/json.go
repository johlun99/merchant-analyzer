// Package exporter serializes feed analysis reports to JSON and Markdown.
package exporter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/johlun99/merchant-analyzer/internal/checker"
)

// AttributeGroup is a named, prioritised group of feed attributes for export.
type AttributeGroup struct {
	Category string
	Items    []Attribute
}

// Attribute is a single feed attribute with its classification tags and product coverage.
type Attribute struct {
	Name     string
	Tags     []string
	Coverage int // 0–100: percentage of products that have this attribute
}

// Report holds all data for an export.
type Report struct {
	URL          string
	FetchTime    time.Duration
	Size         int64
	ProductCount int
	Results      []checker.Result
	Attributes   []AttributeGroup
}

type jsonReport struct {
	URL          string          `json:"url"`
	FetchTimeMs  int64           `json:"fetch_time_ms"`
	SizeBytes    int64           `json:"size_bytes"`
	ProductCount int             `json:"product_count"`
	Results      []jsonResult    `json:"results"`
	Attributes   []jsonAttrGroup `json:"attributes,omitempty"`
}

type jsonAttrGroup struct {
	Category string     `json:"category"`
	Items    []jsonAttr `json:"items"`
}

type jsonAttr struct {
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
	Coverage int      `json:"coverage"`
}

type jsonResult struct {
	Name   string     `json:"name"`
	Status string     `json:"status"`
	Score  *int       `json:"score,omitempty"`
	Items  []jsonItem `json:"items,omitempty"`
}

type jsonItem struct {
	Field    string   `json:"field"`
	Message  string   `json:"message"`
	Count    int      `json:"count,omitempty"`
	Examples []string `json:"examples,omitempty"`
}

// ToJSON serializes a Report to JSON bytes.
func ToJSON(r Report) ([]byte, error) {
	jr := jsonReport{
		URL:          r.URL,
		FetchTimeMs:  r.FetchTime.Milliseconds(),
		SizeBytes:    r.Size,
		ProductCount: r.ProductCount,
	}

	for _, res := range r.Results {
		jr2 := jsonResult{
			Name:   res.Name,
			Status: statusString(res.Status),
			Score:  res.Score,
		}
		for _, item := range res.Items {
			jr2.Items = append(jr2.Items, jsonItem{
				Field:    item.Field,
				Message:  item.Message,
				Count:    item.Count,
				Examples: item.Examples,
			})
		}
		jr.Results = append(jr.Results, jr2)
	}

	for _, g := range r.Attributes {
		if len(g.Items) == 0 {
			continue
		}
		jg := jsonAttrGroup{Category: g.Category}
		for _, a := range g.Items {
			jg.Items = append(jg.Items, jsonAttr(a))
		}
		jr.Attributes = append(jr.Attributes, jg)
	}

	data, err := json.MarshalIndent(jr, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return data, nil
}

func statusString(s checker.Status) string {
	switch s {
	case checker.StatusOK:
		return "ok"
	case checker.StatusWarning:
		return "warning"
	case checker.StatusError:
		return "error"
	case checker.StatusFatal:
		return "fatal"
	}
	return "unknown"
}
