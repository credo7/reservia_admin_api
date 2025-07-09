// Package server provides HTTP server functionality for the Reservia API.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/reservia/api/docs" // Import generated docs
	"github.com/reservia/api/internal/delivery/http/handler"
	"github.com/reservia/api/internal/repository/mongodb"
	cityUseCase "github.com/reservia/api/internal/usecase/city"
	restaurantUseCase "github.com/reservia/api/internal/usecase/restaurant"
	userUseCase "github.com/reservia/api/internal/usecase/user"
	"github.com/reservia/api/pkg/config"
	"github.com/reservia/api/pkg/database"
	"github.com/reservia/api/pkg/logger"
)

// Server represents the HTTP server.
type Server struct {
	config            *config.Config
	db                *database.MongoDB
	logger            logger.Logger
	router            *chi.Mux
	server            *http.Server
	userHandler       *handler.UserHandler
	restaurantHandler *handler.RestaurantHandler
	cityHandler       *handler.CityHandler
	devHandler        *handler.DevHandler
	metricsHandler    *handler.MetricsHandler
	authHandler       *handler.AuthHandler
	roomHandler       *handler.RoomHandler
}

// New creates a new HTTP server instance.
func New(cfg *config.Config, db *database.MongoDB, log logger.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		db:     db,
		logger: log,
	}

	s.setupDependencies()
	s.setupRouter()
	s.setupServer()

	return s, nil
}

// setupDependencies initializes all dependencies using dependency injection.
func (s *Server) setupDependencies() {
	// Initialize repositories
	userRepo := mongodb.NewUserRepository(s.db.Database())
	restaurantRepo := mongodb.NewRestaurantRepository(s.db.Database())
	cityRepo := mongodb.NewCityRepository(s.db.Database())
	// TODO: Add auth and reservation repositories when use cases are implemented
	// authRepo := mongodb.NewAuthRepository(s.db.Database())
	// reservationRepo := mongodb.NewReservationRepository(s.db.Database())

	// Initialize use cases
	userUC := userUseCase.New(userRepo, s.logger)
	restaurantUC := restaurantUseCase.New(restaurantRepo, s.logger)
	cityUC := cityUseCase.NewCityUseCase(cityRepo, s.logger)

	// Initialize handlers
	s.userHandler = handler.NewUserHandler(userUC, s.logger)
	s.restaurantHandler = handler.NewRestaurantHandler(restaurantUC, s.logger)
	s.cityHandler = handler.NewCityHandler(cityUC)
	s.devHandler = handler.NewDevHandler()
	s.metricsHandler = handler.NewMetricsHandler()
	s.authHandler = handler.NewAuthHandler()
	s.roomHandler = handler.NewRoomHandler(restaurantUC)

	// Log successful dependency initialization
	s.logger.Info("Dependencies initialized successfully")
}

// setupRouter configures the HTTP router.
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", s.healthCheck)

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Prometheus metrics endpoint (outside of API versioning)
	r.Get("/metrics", s.metricsHandler.GetMetrics)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Development endpoints
		r.Route("/dev", func(r chi.Router) {
			r.Get("/ping", s.devHandler.Ping)
		})

		// Authentication endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", s.authHandler.Login)
			r.Post("/register", s.authHandler.Register)
			r.Post("/verify", s.authHandler.Verify)
			r.Post("/telegram", s.authHandler.AuthorizeTelegram)
			r.Get("/telegram/{requestID}", s.authHandler.VerifyTelegram)
			r.Get("/by_reservation/{reservationID}", s.authHandler.AuthorizeByReservation)
		})

		// Cities endpoints
		r.Route("/cities", func(r chi.Router) {
			r.Get("/", s.cityHandler.GetCities)
			r.Get("/{name}", s.cityHandler.GetCityByName)
		})

		// User endpoints
		r.Route("/users", func(r chi.Router) {
			r.Post("/", s.userHandler.CreateUser)
			r.Get("/", s.userHandler.ListUsers)
			r.Get("/{id}", s.userHandler.GetUser)
			r.Put("/{id}", s.userHandler.UpdateUser)
			r.Delete("/{id}", s.userHandler.DeleteUser)
		})

		// Restaurant endpoints
		r.Route("/restaurants", func(r chi.Router) {
			r.Post("/", s.restaurantHandler.CreateRestaurant)
			r.Get("/", s.restaurantHandler.ListRestaurants)
			r.Get("/{id}", s.restaurantHandler.GetRestaurant)
			r.Put("/{id}", s.restaurantHandler.UpdateRestaurant)
			r.Delete("/{id}", s.restaurantHandler.DeleteRestaurant)

			// Room endpoints for restaurants
			r.Route("/{restaurantID}/rooms", func(r chi.Router) {
				r.Get("/", s.roomHandler.GetRooms)
				r.Post("/", s.roomHandler.CreateRoom)
				r.Get("/{roomID}", s.roomHandler.GetRoom)
				r.Patch("/{roomID}", s.roomHandler.UpdateRoom)
				r.Delete("/{roomID}", s.roomHandler.DeleteRoom)
			})
		})

		// TODO: Add missing endpoints when implementations are ready:
		// - Employee management endpoints
		// - Element/UI management endpoints
		// - Reservation management endpoints (comprehensive)
	})

	s.router = r
}

// setupServer configures the HTTP server.
func (s *Server) setupServer() {
	s.server = &http.Server{
		Addr:         ":" + s.config.Server.Port,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "port", s.config.Server.Port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// healthCheck handles health check requests.
//
//	@Summary		Health check
//	@Description	Check if the API is running and healthy
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	map[string]string	"API is healthy"
//	@Router			/health [get]
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}
