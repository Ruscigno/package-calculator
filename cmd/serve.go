package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sander-remitly/pack-calc/internal/api"
	"github.com/sander-remitly/pack-calc/internal/cache"
	"github.com/sander-remitly/pack-calc/internal/logger"
	"github.com/sander-remitly/pack-calc/internal/repo"
	"github.com/sander-remitly/pack-calc/internal/web"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI and API server",
	Long:  `Start both the web UI and REST API server together.`,
	Run:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger.Initialize()
	defer logger.Sync()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		logger.Log.Fatal("Failed to create data directory", zap.Error(err))
	}

	// Initialize repository
	repository, err := repo.New(dbPath)
	if err != nil {
		logger.Log.Fatal("Failed to initialize repository", zap.Error(err))
	}
	defer repository.Close()

	// Initialize default pack sizes if none exist
	packSizes, err := repository.GetPackSizes()
	if err != nil {
		logger.Log.Fatal("Failed to get pack sizes", zap.Error(err))
	}
	if len(packSizes) == 0 {
		logger.Log.Info("Initializing default pack sizes...")
		defaultSizes := []int{250, 500, 1000, 2000, 5000}
		if err := repository.SetPackSizes(defaultSizes); err != nil {
			logger.Log.Fatal("Failed to set default pack sizes", zap.Error(err))
		}
	}

	// Initialize cache
	cacheInstance := cache.NewCache()
	defer cacheInstance.Close()

	// Setup API handler
	apiHandler := api.NewHandler(repository, cacheInstance)
	router := apiHandler.SetupRouter()

	// Setup web handler
	webHandler, err := web.NewHandler()
	if err != nil {
		logger.Log.Fatal("Failed to initialize web handler", zap.Error(err))
	}
	webHandler.SetupRoutes(router)

	// Create server
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Log.Info("🚀 Server starting",
			zap.String("url", fmt.Sprintf("http://localhost%s", addr)),
			zap.String("web_ui", fmt.Sprintf("http://localhost%s", addr)),
			zap.String("api", fmt.Sprintf("http://localhost%s/api", addr)),
			zap.String("health", fmt.Sprintf("http://localhost%s/api/health", addr)),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("Server stopped")
}
