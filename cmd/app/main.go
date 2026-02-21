package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/handler"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/ipdata"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/version"
)

var (
	newProcessor   = ipdata.NewProcessor
	newConfig      = config.NewConfig
	newHandler     = handler.NewHandler
	listenAndServe = http.ListenAndServe
	signalNotify   = signal.Notify
	logPrintf      = log.Printf
	logPrintln     = log.Println
	logFatalf      = log.Fatalf
)

func main() {
	fmt.Printf("Starting IP Whitelist by Country server (%s)...\n", version.GetVersion())

	// Create a processor for IP data
	processor := newProcessor()

	// Get configuration
	cfg := newConfig()
	serverAddr := ":" + cfg.ServerPort

	// Pass the configuration to the handler
	h := newHandler(processor, cfg)

	// Register routes
	h.RegisterRoutes()

	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signalNotify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logPrintf("Server started on %s\n", serverAddr)
		if err := listenAndServe(serverAddr, nil); err != nil && err != http.ErrServerClosed {
			logFatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	logPrintln("Shutting down server...")
}
