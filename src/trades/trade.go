package trades

// Trade is a model representing a single trade on binance.
type Trade struct {
	ID     uint64 `json:"t"`
	Symbol string `json:"s"`
	Price  string `json:"p"`
}
