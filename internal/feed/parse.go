// Package feed provides feed fetching and XML parsing.
package feed

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Parse decodes XML feed data into a Feed. Returns an error if the XML is
// malformed or the root element is not properly closed (truncated feed).
func Parse(data []byte) (*Feed, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))

	var products []Product
	var current *Product
	var currentField string
	rootClosed := false
	rootSeen := false

	for {
		tok, err := dec.Token()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("decode xml: %w", err)
			}
			break
		}
		rootClosed, rootSeen, current, currentField = handleToken(tok, rootClosed, rootSeen, &products, current, currentField)
	}

	if !rootSeen {
		return nil, fmt.Errorf("not a recognized feed format (no rss/feed root element found)")
	}
	if !rootClosed {
		return nil, fmt.Errorf("feed is incomplete: root element was not closed")
	}

	return &Feed{
		ProductCount: len(products),
		Products:     products,
	}, nil
}

func handleToken(
	tok xml.Token,
	rootClosed bool,
	rootSeen bool,
	products *[]Product,
	current *Product,
	currentField string,
) (bool, bool, *Product, string) {
	switch t := tok.(type) {
	case xml.StartElement:
		rc, rs, cur, cf := handleStart(t, rootClosed, rootSeen, products, current, currentField)
		return rc, rs, cur, cf
	case xml.EndElement:
		rc, cur, cf := handleEnd(t, rootClosed, products, current, currentField)
		return rc, rootSeen, cur, cf
	case xml.CharData:
		if current != nil && currentField != "" {
			value := strings.TrimSpace(string(t))
			setProductField(current, currentField, value)
		}
	}
	return rootClosed, rootSeen, current, currentField
}

func handleStart(
	t xml.StartElement,
	rootClosed bool,
	rootSeen bool,
	_ *[]Product,
	current *Product,
	_ string,
) (bool, bool, *Product, string) {
	local := strings.ToLower(t.Name.Local)

	if local == "rss" || local == "feed" || local == "channel" {
		return rootClosed, true, current, ""
	}
	if local == "item" || local == "entry" {
		p := Product{Extra: make(map[string]string)}
		return rootClosed, rootSeen, &p, ""
	}
	if current != nil {
		return rootClosed, rootSeen, current, local
	}
	return rootClosed, rootSeen, current, ""
}

func handleEnd(
	t xml.EndElement,
	rootClosed bool,
	products *[]Product,
	current *Product,
	_ string,
) (bool, *Product, string) {
	local := strings.ToLower(t.Name.Local)

	if local == "rss" || local == "feed" {
		return true, current, ""
	}
	if local == "item" || local == "entry" {
		if current != nil {
			*products = append(*products, *current)
		}
		return rootClosed, nil, ""
	}
	return rootClosed, current, ""
}

// setProductField maps a lowercased field name to the appropriate Product field.
func setProductField(p *Product, field, value string) {
	switch field {
	case "id":
		p.ID = value
	case "title":
		p.Title = value
	case "description":
		p.Description = value
	case "price":
		p.Price = value
	case "availability":
		p.Availability = value
	case "link":
		p.Link = value
	case "image_link":
		p.ImageLink = value
	case "brand":
		p.Brand = value
	case "gtin":
		p.GTIN = value
	case "mpn":
		p.MPN = value
	case "condition":
		p.Condition = value
	case "color":
		p.Color = value
	case "size":
		p.Size = value
	case "material":
		p.Material = value
	case "additional_image_link":
		if value != "" {
			p.AdditionalImages = append(p.AdditionalImages, value)
		}
	default:
		if value != "" {
			p.Extra[field] = value
		}
	}
}
