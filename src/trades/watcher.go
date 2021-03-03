package trades

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

	wsConnGuard *sync.RWMutex
	wsConn      *websocket.Conn

	idGuard *sync.RWMutex
	id      uint64
}

// NewWatcher creates a new watcher.
func NewWatcher(logger logger) *Watcher {
	return &Watcher{
		logger:      logger,
		wsConnGuard: &sync.RWMutex{},
		idGuard:     &sync.RWMutex{},
	}
}

func (s *Watcher) conn() (*websocket.Conn, error) {
	s.wsConnGuard.Lock()
	defer s.wsConnGuard.Unlock()

	conn := s.wsConn
	if conn != nil {
		return conn, nil
	}

	url := fmt.Sprintf("%s", wssBase)
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		return nil, fmt.Errorf("failed to dial binance at '%s': %w", url, err)
	}
	s.logger.Debug("new wss connection created")

	s.wsConn = ws
	return ws, nil
}

func (s *Watcher) nextRequestID() uint64 {
	s.idGuard.Lock()
	s.id++
	s.idGuard.Unlock()
	return s.id
}

// Watch returns a new channel with trades for a given symbol.
func (s *Watcher) Watch(ctx context.Context, symbol string, out chan<- *Trade) error {
	ws, err := s.conn()
	if err != nil {
		return err
	}

	requestID := s.nextRequestID()
	if _, err := ws.Write([]byte(fmt.Sprintf(`{
		"method": "SUBSCRIBE",
		"params":["%s@trade"],
		"id": %d}`, strings.ToLower(symbol), requestID,
	))); err != nil {
		return fmt.Errorf("failed to send subscription request: %w", err)
	}

	s.logger.Debug("new watcher for '%s' created", symbol)

	errors := make(chan error)
	go func() {
		var msg = make([]byte, buffSize)
		for {
			size, err := ws.Read(msg)
			if err != nil {
				errors <- fmt.Errorf("failed to read message: %s", err)
				return
			}

			var message struct {
				*Trade
				ID     uint64      `json:"id"`
				Result interface{} `json:"result,omitempty"`
			}

			rawMsg := msg[:size]
			if err := json.Unmarshal(rawMsg, &message); err != nil {
				errors <- fmt.Errorf("failed to unmarshal '%s': %s", string(msg), err)
				return
			}

			switch {
			case message.Trade != nil:
				out <- message.Trade
			case message.ID == requestID && message.Result == nil:
				s.logger.Debug("subscried to '%s@trade'", strings.ToLower(symbol))
			default:
				s.logger.Debug("'%s' watcher ignoring '%s'", symbol, string(rawMsg))
			}

		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Debug("watcher for '%s' stopped", symbol)
		return nil
	case err := <-errors:
		return err
	}
}
