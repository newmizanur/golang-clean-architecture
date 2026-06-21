package model

// Item is hand-written (not generated) so we can add the internal-only
// CurrencyCode field used by the Currency resolver.
type Item struct {
	ID           string
	Name         string
	Sku          string
	Stock        int
	CurrencyCode string // internal only — not a schema field
	CreatedAt    *string
	UpdatedAt    *string
}

type Currency struct {
	Code          string
	Symbol        string
	DecimalPlaces int
}

type ItemPage struct {
	Items []*Item
	Total int
}

type ListItemsInput struct {
	Name *string
	Sku  *string
	Sort *string
	Page int
	Size int
}

type CreateItemInput struct {
	Name     string
	Sku      string
	Currency string
	Stock    int
}
