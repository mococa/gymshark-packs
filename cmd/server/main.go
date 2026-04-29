package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/mococa/gymshark-packs/docs"
	"github.com/mococa/gymshark-packs/internal/calculator"
	"github.com/mococa/gymshark-packs/internal/handler"
	"github.com/mococa/gymshark-packs/internal/middleware"
	"github.com/mococa/gymshark-packs/internal/store"
	"github.com/mococa/gymshark-packs/internal/web"
)

// @title RE Partners Pack Calculator API
// @version 1.0
// @description API for calculating optimal pack sizes for customer orders
// @contact.name Luiz Moureau
// @contact.email luiz@moureau.dev
// @contact.website https://moureau.dev
// @host gymshark-challenge.moureau.dev
// @BasePath /
func main() {
	// Check for healthcheck flag
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		performHealthcheck()
		return
	}

	// Setup structured logging
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Initialize store based on driver
	defaultSizes := []int{250, 500, 1000, 2000, 5000}
	packStore := initializeStore(logger, defaultSizes)
	if closer, ok := packStore.(io.Closer); ok {
		defer closer.Close()
	}

	// Initialize handlers
	calc := calculator.New()
	apiHandler := handler.New(packStore, calc, logger)
	webHandler := web.New(packStore, logger)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(web.StaticFS())))

	// Health check
	r.Get("/healthz", apiHandler.Health)

	// Swagger documentation
	docsHandler := http.StripPrefix("/docs", httpSwagger.Handler(
		httpSwagger.URL("doc.json"),
	))
	r.Get("/docs/", docsHandler.ServeHTTP)
	r.Get("/docs/*", docsHandler.ServeHTTP)

	// API routes
	apiLimiter := middleware.NewRateLimiter(10, 20) // 10 req/s, burst 20
	r.Route("/api", func(r chi.Router) {
		r.Use(apiLimiter.Limit)
		r.Post("/calculate", apiHandler.Calculate)
		r.Get("/pack-sizes", apiHandler.GetPackSizes)
		r.Post("/pack-sizes", apiHandler.AddPackSize)
		r.Delete("/pack-sizes/{size}", apiHandler.DeletePackSize)
	})

	// Web UI routes
	r.Get("/", webHandler.ServeHome)
	r.NotFound(webHandler.ServeNotFound)

	// Start server with graceful shutdown
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Error("graceful shutdown timed out, forcing exit")
			}
		}()

		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			logger.Error("server shutdown failed", "error", err)
		}
		serverStopCtx()
	}()

	logger.Info("starting server", "port", port)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}

	<-serverCtx.Done()
	logger.Info("server stopped")
}

// initializeStore creates the appropriate store based on STORE_DRIVER env var
func initializeStore(logger *slog.Logger, defaultSizes []int) store.PackSizeStore {
	driver := os.Getenv("STORE_DRIVER")

	switch driver {
	case "dynamodb":
		tableName := os.Getenv("DYNAMODB_TABLE")
		if tableName == "" {
			logger.Error("DYNAMODB_TABLE not set, falling back to in-memory")
			return store.NewInMemoryStore(defaultSizes)
		}

		var cfgOpts []func(*config.LoadOptions) error
		if region := os.Getenv("AWS_REGION_CUSTOM"); region != "" {
			cfgOpts = append(cfgOpts, config.WithRegion(region))
		}

		cfg, err := config.LoadDefaultConfig(context.Background(), cfgOpts...)
		if err != nil {
			logger.Error("failed to load AWS config", "error", err)
			logger.Info("falling back to in-memory store")
			return store.NewInMemoryStore(defaultSizes)
		}

		client := dynamodb.NewFromConfig(cfg)
		dynamoStore, err := store.NewDynamoDBStore(client, tableName, defaultSizes)
		if err != nil {
			logger.Error("failed to initialize DynamoDB store", "error", err)
			logger.Info("falling back to in-memory store")
			return store.NewInMemoryStore(defaultSizes)
		}

		logger.Info("using DynamoDB store", "table", tableName)
		return dynamoStore

	case "sqlite":
		dbPath := os.Getenv("SQLITE_PATH")
		if dbPath == "" {
			dbPath = "/tmp/packs.db"
		}

		sqliteStore, err := store.NewSQLiteStore(dbPath, defaultSizes)
		if err != nil {
			logger.Error("failed to initialize SQLite store", "error", err)
			logger.Info("falling back to in-memory store")
			return store.NewInMemoryStore(defaultSizes)
		}

		logger.Info("using SQLite store", "path", dbPath)
		return sqliteStore

	default:
		logger.Info("using in-memory store (no persistence)")
		return store.NewInMemoryStore(defaultSizes)
	}
}

func performHealthcheck() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://localhost:" + port + "/healthz")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}

	os.Exit(0)
}
