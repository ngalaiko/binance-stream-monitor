package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/ngalaiko/binance-stream-monitor/backend/alerts"
	"github.com/ngalaiko/binance-stream-monitor/backend/logger"
	"github.com/ngalaiko/binance-stream-monitor/backend/trades"
)

func main() {
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	var alertListFlag alertList
	flag.Var(&alertListFlag, "alert-on", "Symbol and price to alert on, for example BTCUSDT>51000")
	flag.Parse()

	log := logger.New(logger.Info)
	if *verbose {
		log = logger.New(logger.Debug)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		shutdownCh := make(chan os.Signal)
		signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)
		sig := <-shutdownCh

		log.Info("received %s, stopping", sig)

		cancel()
	}()

	logger := alerts.NewLogger(trades.NewWatcher(log), log)

	if err := logger.Log(ctx, alertListFlag...); err != nil {
		log.Error("error running the application: %s", err)
	}

	log.Info("stopped")
}

type alertList []*alerts.Alert

func (i *alertList) String() string {
	return fmt.Sprint(*i)
}

func (i *alertList) Set(value string) error {
	parts := strings.Split(value, ">")
	if len(parts) != 2 {
		return fmt.Errorf("invalid alert format")
	}

	symbol := parts[0]
	limit, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return fmt.Errorf("failed to parse limit for '%s': %w", symbol, err)
	}

	*i = append(*i, alerts.New(symbol, limit))

	return nil
}
