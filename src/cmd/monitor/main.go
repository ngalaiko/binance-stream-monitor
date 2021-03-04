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

	"github.com/ngalaiko/binance-stream-monitor/src/alerts"
	"github.com/ngalaiko/binance-stream-monitor/src/logger"
	"github.com/ngalaiko/binance-stream-monitor/src/trades"
	"golang.org/x/sync/errgroup"
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

	watcher := trades.NewWatcher(log)
	errGroup, errCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return watcher.Start(errCtx)
	})

	logger := alerts.NewLogger(watcher, log)
	errGroup.Go(func() error {
		return logger.Log(errCtx, alertListFlag...)
	})

	if err := errGroup.Wait(); err != nil {
		log.Error("%s", err)
		return
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
