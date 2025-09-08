// Package database provides database connection management for the Reservia API.
package database

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reservia-admin-api/pkg/logger"
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

// CreateIndexes creates all necessary database indexes at application startup.
func (m *MongoDB) CreateIndexes(ctx context.Context, log logger.Logger) error {
	log.Info("Creating database indexes...")
	
	// Get all existing indexes once for efficiency
	allExistingIndexes, err := m.getAllExistingIndexes(ctx)
	if err != nil {
		return err
	}
	
	// Create indexes for each collection
	if err := m.createFailedAttemptsIndexes(ctx, log, allExistingIndexes); err != nil {
		return err
	}
	
	// Future index creation can be added here:
	// if err := m.createReservationIndexes(ctx, log, allExistingIndexes); err != nil {
	//     return err
	// }
	// if err := m.createEmployeeIndexes(ctx, log, allExistingIndexes); err != nil {
	//     return err
	// }
	
	log.Info("Database indexes created successfully")
	return nil
}

// createFailedAttemptsIndexes creates indexes for the failed_attempts collection.
func (m *MongoDB) createFailedAttemptsIndexes(ctx context.Context, log logger.Logger, existingIndexes map[string]map[string]bool) error {
	collectionName := "failed_attempts"
	collection := m.database.Collection(collectionName)
	
	// Get existing indexes for this collection
	collectionIndexes := existingIndexes[collectionName]
	if collectionIndexes == nil {
		collectionIndexes = make(map[string]bool)
	}
	
	// Define indexes that need to be created
	indexesToCreate := []mongo.IndexModel{}
	
	// TTL index for auto-expiring failed attempts
	ttlIndexName := "expires_at_ttl"
	if !collectionIndexes[ttlIndexName] {
		indexesToCreate = append(indexesToCreate, mongo.IndexModel{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
			Options: &options.IndexOptions{
				ExpireAfterSeconds: &[]int32{0}[0], // Expire at the time specified in expires_at field
				Name:               &ttlIndexName,
			},
		})
	}
	
	// Create indexes if any need to be created
	if len(indexesToCreate) > 0 {
		_, err := collection.Indexes().CreateMany(ctx, indexesToCreate)
		if err != nil {
			log.Error("Failed to create indexes", "collection", collectionName, "error", err)
			return err
		}
		log.Info("Created indexes", "collection", collectionName, "count", len(indexesToCreate))
	} else {
		log.Info("All indexes already exist", "collection", collectionName)
	}
	
	return nil
}

// getAllExistingIndexes returns a map of all existing indexes for all collections.
// Returns map[collectionName]map[indexName]bool
func (m *MongoDB) getAllExistingIndexes(ctx context.Context) (map[string]map[string]bool, error) {
	allIndexes := make(map[string]map[string]bool)
	
	// List of collections to check for indexes
	collections := []string{
		"failed_attempts",
		// Add more collections here as needed:
		// "reservations",
		// "employees",
		// "restaurants",
	}
	
	for _, collectionName := range collections {
		collection := m.database.Collection(collectionName)
		cursor, err := collection.Indexes().List(ctx)
		if err != nil {
			// Collection might not exist yet, skip
			continue
		}
		
		collectionIndexes := make(map[string]bool)
		for cursor.Next(ctx) {
			var index bson.M
			if err := cursor.Decode(&index); err != nil {
				cursor.Close(ctx)
				return nil, err
			}
			
			if name, ok := index["name"].(string); ok {
				collectionIndexes[name] = true
			}
		}
		cursor.Close(ctx)
		
		if err := cursor.Err(); err != nil {
			return nil, err
		}
		
		allIndexes[collectionName] = collectionIndexes
	}
	
	return allIndexes, nil
}
