package main

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/ipdata"
)

type noopProcessor struct{}

func (noopProcessor) GetIPListForCountry(countryCode string) ([]string, error) {
	return []string{}, nil
}

func TestMain_CoversStartupAndFatalPath(t *testing.T) {
	oldMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = oldMux })

	// Save and restore injectable dependencies.
	origNewProcessor := newProcessor
	origNewConfig := newConfig
	origNewHandler := newHandler
	origListenAndServe := listenAndServe
	origSignalNotify := signalNotify
	origLogFatalf := logFatalf

	t.Cleanup(func() {
		newProcessor = origNewProcessor
		newConfig = origNewConfig
		newHandler = origNewHandler
		listenAndServe = origListenAndServe
		signalNotify = origSignalNotify
		logFatalf = origLogFatalf
	})

	newProcessor = func() *ipdata.Processor { return &ipdata.Processor{} }
	newConfig = func() *config.Config { return &config.Config{ServerPort: "0"} }
	// Use the real handler.NewHandler via the existing function pointer.
	_ = newHandler

	var captured chan<- os.Signal
	signalNotify = func(c chan<- os.Signal, _ ...os.Signal) {
		captured = c
	}

	started := make(chan struct{}, 1)
	fatalCalled := make(chan struct{}, 1)

	listenAndServe = func(addr string, handler http.Handler) error {
		started <- struct{}{}
		return errors.New("listen failed")
	}
	logFatalf = func(format string, args ...any) {
		fatalCalled <- struct{}{}
	}

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()

	select {
	case <-started:
		// server goroutine ran
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server goroutine")
	}

	select {
	case <-fatalCalled:
		// fatal path covered
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for logFatalf")
	}

	if captured == nil {
		t.Fatal("expected signal channel to be captured")
	}
	captured <- os.Interrupt

	select {
	case <-done:
		// main returned
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for main to return")
	}
}
