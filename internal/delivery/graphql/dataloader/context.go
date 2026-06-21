package dataloader

import "context"

type contextKey struct{}

// WithLoader stores a fresh CurrencyLoader in the context for this request.
// Call this once per incoming HTTP request (in middleware).
func WithLoader(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, NewCurrencyLoader())
}

// LoaderFrom retrieves the CurrencyLoader from context.
// Returns nil if no loader is present (e.g. in tests without middleware).
func LoaderFrom(ctx context.Context) *CurrencyLoader {
	l, _ := ctx.Value(contextKey{}).(*CurrencyLoader)
	return l
}
