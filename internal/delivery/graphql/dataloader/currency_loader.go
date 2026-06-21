package dataloader

import (
	"context"
	"sync"
	"time"

	"golang-clean-architecture/internal/delivery/graphql/currencyservice"
)

// CurrencyLoader batches currency lookups that arrive within the same
// request-tick into a single GetCurrencies call, eliminating the N+1.
// One instance per incoming GraphQL request — never shared across requests.
type CurrencyLoader struct {
	mu      sync.Mutex
	keys    []string
	results map[string]chan result
	wait    time.Duration
	once    sync.Once
}

type result struct {
	currency currencyservice.Currency
	err      error
}

func NewCurrencyLoader() *CurrencyLoader {
	return &CurrencyLoader{
		results: make(map[string]chan result),
		wait:    2 * time.Millisecond,
	}
}

// Load queues a currency code for batching and blocks until the batch fires.
func (l *CurrencyLoader) Load(ctx context.Context, code string) (currencyservice.Currency, error) {
	l.mu.Lock()
	if _, exists := l.results[code]; !exists {
		l.keys = append(l.keys, code)
		l.results[code] = make(chan result, 1)
	}
	ch := l.results[code]
	l.mu.Unlock()

	// Fire the batch after the wait window (once per loader instance).
	l.once.Do(func() {
		time.AfterFunc(l.wait, func() { l.dispatch(ctx) })
	})

	r := <-ch
	return r.currency, r.err
}

func (l *CurrencyLoader) dispatch(ctx context.Context) {
	l.mu.Lock()
	keys := l.keys
	results := l.results
	l.mu.Unlock()

	// De-duplicate keys before the batch call.
	seen := make(map[string]bool, len(keys))
	unique := make([]string, 0, len(keys))
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			unique = append(unique, k)
		}
	}

	currencies, err := currencyservice.GetCurrencies(ctx, unique)

	for code, ch := range results {
		if err != nil {
			ch <- result{err: err}
		} else {
			ch <- result{currency: currencies[code]}
		}
	}
}
