package trades

// Trade is a model representing a single trade on binance.
type Trade struct {
	Symbol string `json:"s"`
	Price  string `json:"p"`
}
