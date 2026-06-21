package dataloader_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"golang-clean-architecture/internal/delivery/graphql/currencyservice"
	"golang-clean-architecture/internal/delivery/graphql/dataloader"
)

// TestCurrencyLoader_BatchesMultipleCalls verifies that concurrent Load() calls
// for distinct codes are batched into a single GetCurrencies call.
func TestCurrencyLoader_BatchesMultipleCalls(t *testing.T) {
	currencyservice.BatchCallCount.Store(0)

	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	// Use distinct codes: the loader shares one channel per code, so concurrent
	// callers for the same code would contend. Use unique codes here.
	codes := []string{"USD", "SGD", "BDT", "JPY"}
	results := make([]currencyservice.Currency, len(codes))
	errs := make([]error, len(codes))

	// Barrier: ensure all goroutines call Load() concurrently so they all
	// register before the 2 ms dispatch timer fires.
	var ready sync.WaitGroup
	start := make(chan struct{})
	var done sync.WaitGroup

	for i, code := range codes {
		ready.Add(1)
		done.Add(1)
		go func(idx int, c string) {
			defer done.Done()
			ready.Done() // signal that this goroutine is spawned
			<-start      // wait for simultaneous release
			results[idx], errs[idx] = loader.Load(ctx, c)
		}(i, code)
	}
	ready.Wait() // all goroutines are ready at the barrier
	close(start) // release all at once
	done.Wait()

	assert.Equal(t, int64(1), currencyservice.BatchCallCount.Load(), "all loads should be batched into one GetCurrencies call")
	for i, code := range codes {
		assert.NoError(t, errs[i])
		assert.Equal(t, code, results[i].Code)
	}
}

// TestCurrencyLoader_DeduplicatesCodes verifies that 10 concurrent Load() calls
// for the same code "USD" are all batched into a single GetCurrencies call and
// every caller receives the correct result.
func TestCurrencyLoader_DeduplicatesCodes(t *testing.T) {
	currencyservice.BatchCallCount.Store(0)

	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	const n = 10
	results := make([]currencyservice.Currency, n)
	errs := make([]error, n)

	var ready sync.WaitGroup
	start := make(chan struct{})
	var done sync.WaitGroup

	for i := 0; i < n; i++ {
		ready.Add(1)
		done.Add(1)
		go func(idx int) {
			defer done.Done()
			ready.Done()
			<-start
			results[idx], errs[idx] = loader.Load(ctx, "USD")
		}(i)
	}
	ready.Wait()
	close(start)
	done.Wait()

	assert.Equal(t, int64(1), currencyservice.BatchCallCount.Load(), "all 10 loads should be batched into one GetCurrencies call")
	for i := 0; i < n; i++ {
		assert.NoError(t, errs[i])
		assert.Equal(t, "USD", results[i].Code)
		assert.Equal(t, "$", results[i].Symbol)
	}
}

// TestCurrencyLoader_MixedDuplicates verifies that concurrent Load() calls for
// a mix of codes with duplicates are batched into a single GetCurrencies call
// with deduplicated keys, and every caller receives the correct result.
func TestCurrencyLoader_MixedDuplicates(t *testing.T) {
	currencyservice.BatchCallCount.Store(0)

	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	// 6 loads: 3 unique codes, 2 callers each.
	codes := []string{"USD", "USD", "SGD", "SGD", "JPY", "JPY"}
	results := make([]currencyservice.Currency, len(codes))
	errs := make([]error, len(codes))

	var ready sync.WaitGroup
	start := make(chan struct{})
	var done sync.WaitGroup

	for i, code := range codes {
		ready.Add(1)
		done.Add(1)
		go func(idx int, c string) {
			defer done.Done()
			ready.Done()
			<-start
			results[idx], errs[idx] = loader.Load(ctx, c)
		}(i, code)
	}
	ready.Wait()
	close(start)
	done.Wait()

	assert.Equal(t, int64(1), currencyservice.BatchCallCount.Load(), "all 6 loads should be batched into one GetCurrencies call")
	for i, code := range codes {
		assert.NoError(t, errs[i])
		assert.Equal(t, code, results[i].Code)
	}
	// Spot-check symbols.
	assert.Equal(t, "$", results[0].Symbol)
	assert.Equal(t, "$", results[1].Symbol)
	assert.Equal(t, "S$", results[2].Symbol)
	assert.Equal(t, "S$", results[3].Symbol)
}

// TestCurrencyLoader_CorrectValuesReturned verifies the exact field values for a
// known currency loaded in isolation.
func TestCurrencyLoader_CorrectValuesReturned(t *testing.T) {
	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	var wg sync.WaitGroup
	var currency currencyservice.Currency
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		currency, err = loader.Load(ctx, "SGD")
	}()
	wg.Wait()

	assert.NoError(t, err)
	assert.Equal(t, currencyservice.Currency{Code: "SGD", Symbol: "S$", DecimalPlaces: 2}, currency)
}

// TestCurrencyLoader_SecondBatchAfterDispatch verifies that a second wave of
// Load() calls — arriving after the first dispatch has already fired — still
// resolves correctly. This would have hung forever with the old sync.Once
// implementation because the timer was never re-armed.
func TestCurrencyLoader_SecondBatchAfterDispatch(t *testing.T) {
	currencyservice.BatchCallCount.Store(0)

	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	// First batch: load USD, wait for it to resolve.
	var wg sync.WaitGroup
	wg.Add(1)
	var first currencyservice.Currency
	go func() {
		defer wg.Done()
		first, _ = loader.Load(ctx, "USD")
	}()
	wg.Wait()
	assert.Equal(t, "USD", first.Code)

	// Second batch: load SGD after the first dispatch has already fired.
	wg.Add(1)
	var second currencyservice.Currency
	go func() {
		defer wg.Done()
		second, _ = loader.Load(ctx, "SGD")
	}()
	wg.Wait()
	assert.Equal(t, "SGD", second.Code)

	assert.Equal(t, int64(2), currencyservice.BatchCallCount.Load(), "each batch wave should produce one GetCurrencies call")
}

// TestCurrencyLoader_UnknownCodeFallback verifies that a Load() for an
// unknown currency code still returns a non-nil result (the fake service
// returns a fallback Currency rather than omitting the key), so no error
// is expected from the loader layer itself.
func TestCurrencyLoader_UnknownCodeFallback(t *testing.T) {
	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	var currency currencyservice.Currency
	var err error
	go func() {
		defer wg.Done()
		currency, err = loader.Load(ctx, "UNKNOWN")
	}()
	wg.Wait()

	assert.NoError(t, err)
	assert.Equal(t, "UNKNOWN", currency.Code)
	assert.Equal(t, "?", currency.Symbol)
}

// TestWithLoader_StoresAndRetrievesLoader verifies that WithLoader injects a
// non-nil CurrencyLoader into the context and LoaderFrom retrieves it.
func TestWithLoader_StoresAndRetrievesLoader(t *testing.T) {
	ctx := dataloader.WithLoader(context.Background())
	loader := dataloader.LoaderFrom(ctx)
	assert.NotNil(t, loader)
}

// TestLoaderFrom_ReturnsNilWithoutLoader verifies that LoaderFrom returns nil
// when no loader has been stored in the context.
func TestLoaderFrom_ReturnsNilWithoutLoader(t *testing.T) {
	loader := dataloader.LoaderFrom(context.Background())
	assert.Nil(t, loader)
}
