package handler

import (
	"net/http"
	"sync"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/ipdata"
)

// Handler handles HTTP requests for the IP whitelist service
type Handler struct {
	processor ipdata.IPProcessor
	config    *config.Config
	mutex     sync.RWMutex
}

// NewHandler creates a new handler
func NewHandler(processor ipdata.IPProcessor, cfg *config.Config) *Handler {
	return &Handler{
		processor: processor,
		config:    cfg,
	}
}

// RegisterRoutes registers the HTTP routes for the handler
func (h *Handler) RegisterRoutes() {
	h.RegisterRoutesOn(http.DefaultServeMux)
}

// RegisterRoutesOn registers the HTTP routes for the handler on the provided mux.
func (h *Handler) RegisterRoutesOn(mux *http.ServeMux) {
	mux.HandleFunc("/get", h.getIpListHandler)
}

// getIpListHandler handles requests to get IP list for a country
func (h *Handler) getIpListHandler(w http.ResponseWriter, r *http.Request) {
	// Validate request method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	country := r.URL.Query().Get("country")
	auth := r.URL.Query().Get("auth")

	// Validate parameters
	if country == "" {
		http.Error(w, "Missing country parameter", http.StatusBadRequest)
		return
	}

	// Only check authentication if an AuthToken is configured
	if h.config.AuthToken != "" && (auth == "" || auth != h.config.AuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Process the request
	ipList, err := h.processor.GetIPListForCountry(country)
	if err != nil {
		http.Error(w, "Error processing request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "text/plain")

	// Write the response
	for _, ip := range ipList {
		w.Write([]byte(ip + "\n"))
	}
}
