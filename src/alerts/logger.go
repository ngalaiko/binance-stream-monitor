package alerts

import (
	"context"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/ngalaiko/binance-stream-monitor/src/trades"
)

type logger interface {
	Warn(string, ...interface{})
	Error(string, ...interface{})
}

type tradesWatcher interface {
	Watch(context.Context, string, chan<- *trades.Trade) error
}

// Logger logs alerts that are triggered by observed trades.
type Logger struct {
	watcher tradesWatcher
	logger  logger
}

// NewLogger returns a new logger instance.
func NewLogger(watcher tradesWatcher, logger logger) *Logger {
	return &Logger{
		watcher: watcher,
		logger:  logger,
	}
}

// Log logs alert using logger when it's triggered.
func (l *Logger) Log(ctx context.Context, alerts ...*Alert) error {
	alertsBySymbol := map[string][]*Alert{}
	for _, alert := range alerts {
		aa, found := alertsBySymbol[alert.symbol]
		if !found {
			aa = []*Alert{}
		}
		aa = append(aa, alert)
		alertsBySymbol[alert.symbol] = aa
	}

	errGroup, errCtx := errgroup.WithContext(ctx)
	for symbol, alerts := range alertsBySymbol {
		alerts := alerts
		symbol := symbol
		errGroup.Go(func() error {
			return l.log(errCtx, symbol, alerts...)
		})
	}

	return errGroup.Wait()
}

func (l *Logger) log(ctx context.Context, symbol string, alerts ...*Alert) error {
	trades := make(chan *trades.Trade)

	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				l.logger.Error("panic: %s", r)
			}
		}()
		return l.watcher.Watch(ctx, symbol, trades)
	})

	go func() {
		errGroup.Wait()
		close(trades)
	}()

	for trade := range trades {
		tradePrice, err := strconv.ParseFloat(trade.Price, 64)
		if err != nil {
			return err
		}

		for _, alert := range alerts {
			if !strings.EqualFold(alert.symbol, trade.Symbol) {
				continue
			}
			if tradePrice > alert.limit {
				l.logger.Warn("%s price exceeded %f: %f", alert.symbol, alert.limit, tradePrice)
			}
		}
	}

	return errGroup.Wait()
}
