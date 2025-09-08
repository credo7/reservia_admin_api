// Package main provides the entry point for the Reservia API server.
//
//	@title			Reservia API
//	@version		1.0
//	@description	A restaurant reservation management system API built with clean architecture.
//	@termsOfService	http://swagger.io/terms/
//
//	@contact.name	Reservia API Support
//	@contact.url	http://www.reservia.com/support
//	@contact.email	support@reservia.com
//
//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT
//
//	@host		localhost:8080
//	@BasePath	/api/admin
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.
package main

import (
	"context"
	"reservia-admin-api/internal/server"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reservia-admin-api/pkg/config"
	"reservia-admin-api/pkg/database"
	"reservia-admin-api/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.New()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", "error", err)
	}

	// Initialize database
	db, err := database.NewMongoDB(cfg.Database.URL)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()
	
	// Create database indexes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err = db.CreateIndexes(ctx, log); err != nil {
		cancel()
		log.Fatal("Failed to create database indexes", "error", err)
	}
	cancel()

	// Initialize HTTP server
	srv, err := server.New(cfg, db, log)
	if err != nil {
		log.Fatal("Failed to create server", "error", err)
	}

	// Start server
	go func() {
		log.Info("Starting server", "port", cfg.Server.Port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}

// Test comment for pre-commit hook
// Updated comment
