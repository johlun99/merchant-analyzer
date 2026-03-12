// Package googlespec validates Google Merchant Center feed attribute presence and format.
package googlespec

import (
	"regexp"
	"strings"

	"github.com/johlun99/merchant-analyzer/internal/feed"
)

//nolint:gochecknoglobals
var (
	priceRe = regexp.MustCompile(`^\d+(\.\d{1,4})? [A-Z]{3}$`)
	gtinRe  = regexp.MustCompile(`^(\d{8}|\d{12}|\d{13}|\d{14})$`)
)

type fieldRule struct {
	field         string
	required      bool
	checkPresence bool
	get           func(*feed.Product) string
	validate      func(string) bool
	hint          string
}

//nolint:gochecknoglobals
var rules = []fieldRule{
	// Required fields
	{
		field: "id", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.ID },
		validate: func(v string) bool { return len(v) <= 50 },
		hint:     "max 50 characters",
	},
	{
		field: "title", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.Title },
		validate: func(v string) bool { return len(v) <= 150 },
		hint:     "max 150 characters",
	},
	{
		field: "description", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.Description },
		validate: func(v string) bool { return len(v) <= 5000 },
		hint:     "max 5000 characters",
	},
	{
		field: "link", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.Link },
		validate: isURL,
		hint:     "http:// or https:// URL",
	},
	{
		field: "image_link", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.ImageLink },
		validate: isURL,
		hint:     "http:// or https:// URL",
	},
	{
		field: "price", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.Price },
		validate: func(v string) bool { return priceRe.MatchString(v) },
		hint:     "9.99 SEK",
	},
	{
		field: "availability", required: true, checkPresence: true,
		get:      func(p *feed.Product) string { return p.Availability },
		validate: isOneOf("in stock", "out of stock", "preorder", "backorder"),
		hint:     "in stock, out of stock, preorder, or backorder",
	},
	// Recommended — presence required
	{
		field: "brand", required: false, checkPresence: true,
		get: func(p *feed.Product) string { return p.Brand },
	},
	{
		field: "google_product_category", required: false, checkPresence: true,
		get: func(p *feed.Product) string { return p.Extra["google_product_category"] },
	},
	// Recommended — format validated if present
	{
		field: "condition", required: false, checkPresence: false,
		get:      func(p *feed.Product) string { return p.Condition },
		validate: isOneOf("new", "refurbished", "used"),
		hint:     "new, refurbished, or used",
	},
	{
		field: "gtin", required: false, checkPresence: false,
		get:      func(p *feed.Product) string { return p.GTIN },
		validate: func(v string) bool { return gtinRe.MatchString(v) },
		hint:     "8, 12, 13, or 14 digit number",
	},
	{
		field: "age_group", required: false, checkPresence: false,
		get:      func(p *feed.Product) string { return p.Extra["age_group"] },
		validate: isOneOf("newborn", "infant", "toddler", "kids", "adult"),
		hint:     "newborn, infant, toddler, kids, or adult",
	},
	{
		field: "gender", required: false, checkPresence: false,
		get:      func(p *feed.Product) string { return p.Extra["gender"] },
		validate: isOneOf("male", "female", "unisex"),
		hint:     "male, female, or unisex",
	},
}

func isURL(v string) bool {
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

func isOneOf(values ...string) func(string) bool {
	return func(v string) bool {
		for _, allowed := range values {
			if v == allowed {
				return true
			}
		}
		return false
	}
}
