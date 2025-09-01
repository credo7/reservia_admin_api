// Package database provides database connection management for the Reservia API.
package database

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB represents a MongoDB database connection.
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection.
func NewMongoDB(uri string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client options for better resilience to network disruptions
	clientOptions := options.Client().ApplyURI(uri).
		SetMaxPoolSize(100).                // Maximum number of connections in the pool
		SetMinPoolSize(5).                  // Minimum number of connections in the pool
		SetMaxConnIdleTime(30 * time.Second). // Close connections after 30 seconds of inactivity
		SetServerSelectionTimeout(5 * time.Second). // Timeout for server selection
		SetSocketTimeout(30 * time.Second). // Socket timeout
		SetConnectTimeout(10 * time.Second). // Connection timeout
		SetHeartbeatInterval(10 * time.Second). // Ping interval to detect connection issues
		SetRetryWrites(true).               // Enable retryable writes
		SetRetryReads(true)                 // Enable retryable reads

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	// Extract database name from URI or use default
	dbName := extractDatabaseNameFromURI(uri)
	if dbName == "" {
		dbName = "reservia" // fallback to default
	}

	return &MongoDB{
		client:   client,
		database: client.Database(dbName),
	}, nil
}

// Collection returns a collection from the database.
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

// Close closes the database connection.
func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// Database returns the underlying database instance.
func (m *MongoDB) Database() *mongo.Database {
	return m.database
}

// Client returns the underlying client instance.
func (m *MongoDB) Client() *mongo.Client {
	return m.client
}

// extractDatabaseNameFromURI extracts the database name from a MongoDB URI.
// Example: mongodb://user:pass@host:port/dbname?authSource=admin -> "dbname"
func extractDatabaseNameFromURI(uri string) string {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	// Remove leading slash from path
	path := strings.TrimPrefix(parsedURI.Path, "/")

	// If no path or just "/", return empty
	if path == "" {
		return ""
	}

	// Split by "/" and take the first part (database name)
	parts := strings.Split(path, "/")
	return parts[0]
}
