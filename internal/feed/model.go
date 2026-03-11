// Package feed provides HTTP fetching and XML parsing of merchant product feeds.
package feed

import "time"

// Feed holds the parsed contents and fetch metadata of a merchant feed.
type Feed struct {
	URL          string
	FetchTime    time.Duration
	Size         int64
	ProductCount int
	Products     []Product
	Raw          []byte // original XML bytes, used for structural validation
}

// Product represents a single product entry in a merchant feed.
// Known fields are mapped to named fields; all other attributes land in Extra
// with namespace prefixes stripped and names lowercased.
type Product struct {
	ID               string
	Title            string
	Description      string
	Price            string
	Availability     string
	Link             string
	ImageLink        string
	Brand            string
	GTIN             string
	MPN              string
	Condition        string
	Color            string
	Size             string
	Material         string
	AdditionalImages []string
	Extra            map[string]string
}
