package views

// attrMeta classifies a feed attribute by its role in Google Merchant Center and AI readiness.
type attrMeta struct {
	GoogleRequired    bool
	GoogleRecommended bool
	GoogleSupported   bool
	AIReady           bool
}

// knownAttrs is the canonical registry of known feed attributes.
//
//nolint:gochecknoglobals
var knownAttrs = map[string]attrMeta{
	// Required by Google + AI UCP
	"id":           {GoogleRequired: true, AIReady: true},
	"title":        {GoogleRequired: true, AIReady: true},
	"description":  {GoogleRequired: true, AIReady: true},
	"link":         {GoogleRequired: true, AIReady: true},
	"image_link":   {GoogleRequired: true, AIReady: true},
	"price":        {GoogleRequired: true, AIReady: true},
	"availability": {GoogleRequired: true, AIReady: true},
	// Recommended by Google
	"brand":                   {GoogleRecommended: true, AIReady: true},
	"google_product_category": {GoogleRecommended: true},
	"mpn":                     {GoogleRecommended: true, AIReady: true},
	"additional_image_link":   {GoogleRecommended: true, AIReady: true},
	"product_type":            {GoogleRecommended: true},
	// Supported by Google (format-validated or apparel attributes)
	"condition":               {GoogleSupported: true, AIReady: true},
	"gtin":                    {GoogleSupported: true, AIReady: true},
	"age_group":               {GoogleSupported: true},
	"gender":                  {GoogleSupported: true},
	"color":                   {GoogleSupported: true, AIReady: true},
	"size":                    {GoogleSupported: true, AIReady: true},
	"material":                {GoogleSupported: true, AIReady: true},
	"sale_price":              {GoogleSupported: true},
	"size_type":               {GoogleSupported: true},
	"size_system":             {GoogleSupported: true},
	"adult":                   {GoogleSupported: true},
	"is_bundle":               {GoogleSupported: true},
	"multipack":               {GoogleSupported: true},
	"energy_efficiency_class": {GoogleSupported: true},
}
