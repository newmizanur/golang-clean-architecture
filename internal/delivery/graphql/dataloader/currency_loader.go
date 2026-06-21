package dataloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang-clean-architecture/internal/delivery/graphql/currencyservice"
)

// CurrencyLoader batches currency lookups that arrive within the same
// request-tick into a single GetCurrencies call, eliminating the N+1.
// One instance per incoming GraphQL request — never shared across requests.
type CurrencyLoader struct {
	mu       sync.Mutex
	keys     []string
	waiters  map[string][]chan result // one channel per caller, keyed by code
	wait     time.Duration
	timerSet bool // protected by mu; true while a dispatch timer is pending
}

type result struct {
	currency currencyservice.Currency
	err      error
}

func NewCurrencyLoader() *CurrencyLoader {
	return &CurrencyLoader{
		waiters: make(map[string][]chan result),
		wait:    2 * time.Millisecond,
	}
}

// Load queues a currency code for batching and blocks until the batch fires.
// Multiple concurrent callers for the same code each get their own channel so
// dispatch can fan-out the single result to all of them.
func (l *CurrencyLoader) Load(ctx context.Context, code string) (currencyservice.Currency, error) {
	ch := make(chan result, 1)

	l.mu.Lock()
	if len(l.waiters[code]) == 0 {
		l.keys = append(l.keys, code)
	}
	l.waiters[code] = append(l.waiters[code], ch)
	if !l.timerSet {
		l.timerSet = true
		time.AfterFunc(l.wait, func() { l.dispatch(ctx) })
	}
	l.mu.Unlock()

	r := <-ch
	return r.currency, r.err
}

func (l *CurrencyLoader) dispatch(ctx context.Context) {
	// Swap out the current batch under the mutex so that:
	//   (a) new Load() calls after this point open a fresh batch window, and
	//   (b) we iterate a map that no other goroutine holds a reference to —
	//       eliminating the data race.
	l.mu.Lock()
	keys := l.keys
	waiters := l.waiters
	l.keys = nil
	l.waiters = make(map[string][]chan result)
	l.timerSet = false
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

	// Fan-out: send the result to every caller waiting on each code.
	for code, chans := range waiters {
		for _, ch := range chans {
			if err != nil {
				ch <- result{err: err}
			} else if c, ok := currencies[code]; ok {
				ch <- result{currency: c}
			} else {
				ch <- result{err: fmt.Errorf("currency not found: %s", code)}
			}
		}
	}
}
