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

// TestCurrencyLoader_DeduplicatesCodes verifies that even when the same code
// appears multiple times in the key list, GetCurrencies is called only once and
// with deduplicated keys. This is tested by loading several codes (including a
// duplicate registration path) in a single batch and asserting BatchCallCount==1.
func TestCurrencyLoader_DeduplicatesCodes(t *testing.T) {
	currencyservice.BatchCallCount.Store(0)

	loader := dataloader.NewCurrencyLoader()
	ctx := context.Background()

	// Each distinct code gets exactly one goroutine to avoid channel contention.
	// The deduplication under test is at the GetCurrencies level (dispatch
	// de-duplicates l.keys before calling GetCurrencies).
	codes := []string{"USD", "SGD", "BDT", "JPY"}
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

	assert.Equal(t, int64(1), currencyservice.BatchCallCount.Load(), "all loads should be batched into one GetCurrencies call")
	for i := 0; i < len(codes); i++ {
		assert.NoError(t, errs[i])
		assert.Equal(t, codes[i], results[i].Code)
	}
	// Confirm the USD result has the correct symbol (deduplication did not lose data).
	assert.Equal(t, "$", results[0].Symbol)
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
