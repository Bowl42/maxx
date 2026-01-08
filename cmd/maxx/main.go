package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Bowl42/maxx-next/internal/adapter/client"
	_ "github.com/Bowl42/maxx-next/internal/adapter/provider/antigravity" // Register antigravity adapter
	_ "github.com/Bowl42/maxx-next/internal/adapter/provider/custom"      // Register custom adapter
	"github.com/Bowl42/maxx-next/internal/executor"
	"github.com/Bowl42/maxx-next/internal/handler"
	"github.com/Bowl42/maxx-next/internal/repository/cached"
	"github.com/Bowl42/maxx-next/internal/repository/sqlite"
	"github.com/Bowl42/maxx-next/internal/router"
)

// getDefaultDBPath returns the default database path (~/.config/maxx/maxx.db)
func getDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir is unavailable
		return "maxx.db"
	}
	return filepath.Join(homeDir, ".config", "maxx", "maxx.db")
}

func main() {
	// Get default database path
	defaultDBPath := getDefaultDBPath()

	// Parse flags
	addr := flag.String("addr", ":9880", "Server address")
	dbPath := flag.String("db", defaultDBPath, "SQLite database path")
	flag.Parse()

	// Ensure database directory exists
	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory %s: %v", dbDir, err)
	}

	// Initialize database
	db, err := sqlite.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create repositories
	providerRepo := sqlite.NewProviderRepository(db)
	routeRepo := sqlite.NewRouteRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	retryConfigRepo := sqlite.NewRetryConfigRepository(db)
	routingStrategyRepo := sqlite.NewRoutingStrategyRepository(db)
	proxyRequestRepo := sqlite.NewProxyRequestRepository(db)
	attemptRepo := sqlite.NewProxyUpstreamAttemptRepository(db)
	settingRepo := sqlite.NewSystemSettingRepository(db)

	// Create cached repositories
	cachedProviderRepo := cached.NewProviderRepository(providerRepo)
	cachedRouteRepo := cached.NewRouteRepository(routeRepo)
	cachedRetryConfigRepo := cached.NewRetryConfigRepository(retryConfigRepo)
	cachedRoutingStrategyRepo := cached.NewRoutingStrategyRepository(routingStrategyRepo)
	cachedSessionRepo := cached.NewSessionRepository(sessionRepo)

	// Load cached data
	if err := cachedProviderRepo.Load(); err != nil {
		log.Printf("Warning: Failed to load providers cache: %v", err)
	}
	if err := cachedRouteRepo.Load(); err != nil {
		log.Printf("Warning: Failed to load routes cache: %v", err)
	}
	if err := cachedRetryConfigRepo.Load(); err != nil {
		log.Printf("Warning: Failed to load retry configs cache: %v", err)
	}
	if err := cachedRoutingStrategyRepo.Load(); err != nil {
		log.Printf("Warning: Failed to load routing strategies cache: %v", err)
	}

	// Create router
	r := router.NewRouter(cachedRouteRepo, cachedProviderRepo, cachedRoutingStrategyRepo, cachedRetryConfigRepo)

	// Create WebSocket hub
	wsHub := handler.NewWebSocketHub()

	// Setup log output to broadcast via WebSocket
	logWriter := handler.NewWebSocketLogWriter(wsHub, os.Stdout)
	log.SetOutput(logWriter)

	// Create executor
	exec := executor.NewExecutor(r, proxyRequestRepo, attemptRepo, cachedRetryConfigRepo, wsHub)

	// Create client adapter
	clientAdapter := client.NewAdapter()

	// Create handlers
	proxyHandler := handler.NewProxyHandler(clientAdapter, exec, cachedSessionRepo)
	adminHandler := handler.NewAdminHandler(
		cachedProviderRepo,
		cachedRouteRepo,
		projectRepo,
		cachedSessionRepo,
		cachedRetryConfigRepo,
		cachedRoutingStrategyRepo,
		proxyRequestRepo,
		settingRepo,
		*addr,
	)

	// Setup routes
	mux := http.NewServeMux()

	// Admin API routes
	mux.Handle("/admin/", adminHandler)

	// Proxy routes - catch all AI API endpoints
	// Claude API
	mux.Handle("/v1/messages", proxyHandler)
	// OpenAI API
	mux.Handle("/v1/chat/completions", proxyHandler)
	// Codex API
	mux.Handle("/v1/responses", proxyHandler)
	// Gemini API (Google AI Studio style)
	mux.Handle("/v1beta/models/", proxyHandler)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHub.HandleWebSocket)

	// Serve static files (Web UI) - must be last (default route)
	staticHandler := handler.NewStaticHandler()
	mux.Handle("/", staticHandler)

	// Wrap with logging middleware
	loggedMux := handler.LoggingMiddleware(mux)

	// Start server
	log.Printf("Starting maxx-next server on %s", *addr)
	log.Printf("Database: %s", *dbPath)
	log.Printf("Admin API: http://localhost%s/admin/", *addr)
	log.Printf("WebSocket: ws://localhost%s/ws", *addr)
	log.Printf("Proxy endpoints:")
	log.Printf("  Claude: http://localhost%s/v1/messages", *addr)
	log.Printf("  OpenAI: http://localhost%s/v1/chat/completions", *addr)
	log.Printf("  Codex:  http://localhost%s/v1/responses", *addr)
	log.Printf("  Gemini: http://localhost%s/v1beta/models/{model}:generateContent", *addr)

	if err := http.ListenAndServe(*addr, loggedMux); err != nil {
		log.Printf("Server error: %v", err)
		os.Exit(1)
	}
}
