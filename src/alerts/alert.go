package alerts

// Alert describes a simple symbol trade price alert.
type Alert struct {
	symbol string
	limit  float64
}

// New creates new alert.
func New(symbol string, limit float64) *Alert {
	return &Alert{
		symbol: symbol,
		limit:  limit,
	}
}
