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

	watchersGuard *sync.RWMutex
	watchers      map[string][]chan<- *Trade

	watchRequests chan string
}

// NewWatcher creates a new watcher.
func NewWatcher(logger logger) *Watcher {
	return &Watcher{
		logger:        logger,
		wsConnGuard:   &sync.RWMutex{},
		idGuard:       &sync.RWMutex{},
		watchers:      map[string][]chan<- *Trade{},
		watchersGuard: &sync.RWMutex{},
		watchRequests: make(chan string),
	}
}

// Start starts the watcher
func (s *Watcher) Start(ctx context.Context) error {
	ws, err := s.conn()
	if err != nil {
		return err
	}

	errors := make(chan error)
	trades := make(chan *Trade)
	var msg = make([]byte, buffSize)
	go func() {
		for {
			size, err := ws.Read(msg)
			if err != nil {
				errors <- fmt.Errorf("failed to read from ws: %w", err)
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
				trades <- message.Trade
			default:
				s.logger.Debug("ignoring '%s'", string(rawMsg))
			}
		}
	}()

	for {
		select {
		case symbol := <-s.watchRequests:
			s.logger.Debug("subscribing to %s", symbol)
			requestID := s.nextRequestID()
			if _, err := ws.Write([]byte(fmt.Sprintf(`{
			"method": "SUBSCRIBE",
			"params":["%s@trade"],
			"id": %d}`, strings.ToLower(symbol), requestID,
			))); err != nil {
				return fmt.Errorf("failed to send subscription request: %w", err)
			}
		case <-ctx.Done():
			return ws.Close()
		case err := <-errors:
			_ = ws.Close()
			return err
		case trade := <-trades:
			for _, out := range s.getWatcher(strings.ToLower(trade.Symbol)) {
				tCopy := *trade
				out <- &tCopy
			}
		}
	}
}

func (s *Watcher) registerWatcher(symbol string, out chan<- *Trade) {
	s.watchRequests <- symbol

	s.watchersGuard.RLock()
	ww, exists := s.watchers[symbol]
	s.watchersGuard.RUnlock()

	if !exists {
		ww = []chan<- *Trade{}
	}

	ww = append(ww, out)

	s.watchersGuard.Lock()
	s.watchers[symbol] = ww
	s.watchersGuard.Unlock()
}

func (s *Watcher) getWatcher(symbol string) []chan<- *Trade {
	s.watchersGuard.RLock()
	ww := s.watchers[symbol]
	s.watchersGuard.RUnlock()

	return ww
}

// Watch subscribes to symbol trades, and writes them into the channel.
func (s *Watcher) Watch(ctx context.Context, symbol string, out chan<- *Trade) error {
	s.registerWatcher(strings.ToLower(symbol), out)

	s.logger.Debug("new watcher for '%s' created", symbol)

	select {
	case <-ctx.Done():
		s.logger.Debug("watcher for '%s' stopped", symbol)
		return nil
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
