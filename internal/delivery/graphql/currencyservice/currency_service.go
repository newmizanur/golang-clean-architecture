package currencyservice

import (
	"context"
	"sync/atomic"
	"time"
)

type Currency struct {
	Code          string
	Symbol        string
	DecimalPlaces int
}

var fakeCurrencyDB = map[string]Currency{
	"USD": {Code: "USD", Symbol: "$", DecimalPlaces: 2},
	"SGD": {Code: "SGD", Symbol: "S$", DecimalPlaces: 2},
	"BDT": {Code: "BDT", Symbol: "Tk", DecimalPlaces: 2},
	"JPY": {Code: "JPY", Symbol: "¥", DecimalPlaces: 0},
}

// CallCount tracks individual GetCurrency calls (for N+1 demo).
var CallCount atomic.Int64

// GetCurrency simulates a single-row lookup with 5ms latency.
// Called once PER item in the naive resolver — the N+1 problem.
func GetCurrency(ctx context.Context, code string) (Currency, error) {
	CallCount.Add(1)
	time.Sleep(5 * time.Millisecond)
	c, ok := fakeCurrencyDB[code]
	if !ok {
		return Currency{Code: code, Symbol: "?", DecimalPlaces: 2}, nil
	}
	return c, nil
}

// BatchCallCount tracks batched GetCurrencies calls (for DataLoader demo).
var BatchCallCount atomic.Int64

// GetCurrencies simulates ONE batched lookup regardless of how many codes are
// requested — used by the DataLoader to collapse N calls into 1.
func GetCurrencies(ctx context.Context, codes []string) (map[string]Currency, error) {
	BatchCallCount.Add(1)
	time.Sleep(5 * time.Millisecond)
	result := make(map[string]Currency, len(codes))
	for _, c := range codes {
		if cur, ok := fakeCurrencyDB[c]; ok {
			result[c] = cur
		} else {
			result[c] = Currency{Code: c, Symbol: "?", DecimalPlaces: 2}
		}
	}
	return result, nil
}
