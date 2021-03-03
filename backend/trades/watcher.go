package trades

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/websocket"
)

const (
	origin   = "https://stream.binance.com"
	wssBase  = "wss://stream.binance.com:9443/ws"
	buffSize = 2048
)

type logger interface {
	Debug(string, ...interface{})
}

// Watcher listense to binance events.
type Watcher struct {
	logger logger
}

// NewWatcher creates a new watcher.
func NewWatcher(logger logger) *Watcher {
	return &Watcher{
		logger: logger,
	}
}

// Watch returns a new channel with trades for a given symbol.
func (s *Watcher) Watch(ctx context.Context, symbol string, out chan<- *Trade) error {
	url := fmt.Sprintf("%s/%s@trade", wssBase, strings.ToLower(symbol))
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		return fmt.Errorf("failed to dial binance at '%s': %w", url, err)
	}

	s.logger.Debug("new watcher for '%s' created", symbol)

	var msg = make([]byte, buffSize)
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("watcher for '%s' stopped", symbol)
			return nil
		default:
			size, err := ws.Read(msg)
			if err != nil {
				return fmt.Errorf("failed to read message: %s", err)
			}

			var message struct {
				*Trade
			}

			rawMsg := msg[:size]
			if err := json.Unmarshal(rawMsg, &message); err != nil {
				return fmt.Errorf("failed to unmarshal '%s': %s", string(msg), err)
			}

			if message.Trade != nil {
				out <- message.Trade
			} else {
				s.logger.Debug("'%s' watcher ignoring '%s'", symbol, string(rawMsg))
			}
		}
	}
}
