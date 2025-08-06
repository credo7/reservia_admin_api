// Package server provides HTTP server functionality for the Reservia API.
package server

import (
	"context"
	handler "github.com/reservia/api/internal/handler"
	"github.com/reservia/api/internal/middleware"
	"github.com/reservia/api/internal/service"
	"net/http"
	"time"

	chi "github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/reservia/api/docs" // Import generated docs
	"github.com/reservia/api/internal/repository/mongodb"
	"github.com/reservia/api/pkg/config"
	"github.com/reservia/api/pkg/database"
	"github.com/reservia/api/pkg/logger"
	"github.com/reservia/api/pkg/rabbitmq"
)

// Server represents the HTTP server.
type Server struct {
	config             *config.Config
	db                 *database.MongoDB
	logger             logger.Logger
	router             *chi.Mux
	server             *http.Server
	authMiddleware     *middleware.AuthMiddleware
	employeeHandler    *handler.EmployeeHandler
	restaurantHandler  *handler.RestaurantHandler
	cityHandler        *handler.CityHandler
	devHandler         *handler.DevHandler
	metricsHandler     *handler.MetricsHandler
	authHandler        *handler.AuthHandler
	roomHandler        *handler.RoomHandler
	reservationHandler *handler.ReservationHandler
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
	employeeRepo := mongodb.NewEmployeeRepository(s.db.Database())
	restaurantRepo := mongodb.NewRestaurantRepository(s.db.Database())
	cityRepo := mongodb.NewCityRepository(s.db.Database())
	authRepo := mongodb.NewAuthRepository(s.db.Database())
	reservationRepo := mongodb.NewReservationRepository(s.db.Database())

	// Initialize RabbitMQ producer for email service
	var emailProducer *rabbitmq.Producer
	if s.config.RabbitMQ.URI != "" {
		rabbitConfig := rabbitmq.Config{
			URI:       s.config.RabbitMQ.URI,
			QueueName: s.config.RabbitMQ.QueueName,
		}

		var err error
		emailProducer, err = rabbitmq.NewProducer(rabbitConfig, s.logger)
		if err != nil {
			s.logger.Error("Failed to initialize RabbitMQ producer", "error", err)
			s.logger.Warn("Email notifications will be disabled")
			emailProducer = nil
		} else {
			s.logger.Info("RabbitMQ producer initialized successfully", "queue", s.config.RabbitMQ.QueueName)
		}
	} else {
		s.logger.Warn("RabbitMQ URI not configured, email notifications will be disabled")
	}

	// Initialize services
	employeeService := service.NewEmployeeService(employeeRepo, s.logger)
	cityService := service.NewCityService(cityRepo, s.logger)
	restaurantService := service.NewRestaurantService(restaurantRepo, employeeRepo, cityService, s.logger)
	authService := service.NewAuthService(authRepo, employeeRepo, emailProducer, s.config.RabbitMQ.QueueName, s.config, s.logger)
	reservationService := service.NewReservationService(reservationRepo, restaurantRepo, s.logger)

	// Initialize middleware
	s.authMiddleware = middleware.NewAuthMiddleware(authService, employeeService, s.logger)

	// Initialize handlers
	s.employeeHandler = handler.NewEmployeeHandler(employeeService, authService, s.logger)
	s.restaurantHandler = handler.NewRestaurantHandler(restaurantService, reservationService, authService, s.logger)
	s.cityHandler = handler.NewCityHandler(cityService, s.logger)
	s.devHandler = handler.NewDevHandler()
	s.metricsHandler = handler.NewMetricsHandler()
	s.authHandler = handler.NewAuthHandler(authService, s.logger)
	s.roomHandler = handler.NewRoomHandler(restaurantService, authService, s.logger)
	s.reservationHandler = handler.NewReservationHandler(reservationService, authService, s.logger)

	// Log successful dependency initialization
	s.logger.Info("Dependencies initialized successfully")
}

// setupRouter configures the HTTP router.
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	
	// Custom logging middleware that excludes health checks
	loggingMiddleware := middleware.NewLoggingMiddleware(s.logger)
	r.Use(loggingMiddleware.Handler)
	
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))

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
	r.Route("/api/admin", func(r chi.Router) {
		// Development endpoints
		r.Route("/dev", func(r chi.Router) {
			r.Get("/ping", s.devHandler.Ping)
		})

		// Authentication endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", s.authHandler.Login)
			r.Post("/register", s.authHandler.Register)
			r.Post("/verify", s.authHandler.Verify)
			r.Get("/register/employee", s.authHandler.RegisterEmployee)
			r.Get("/tg", s.authHandler.AuthorizeTelegram)
			r.Get("/tg/{requestID}", s.authHandler.VerifyTelegram)
		})

		// Cities endpoints
		r.Route("/cities", func(r chi.Router) {
			r.Get("/", s.cityHandler.GetCities)
			r.Get("/{name}", s.cityHandler.GetCityByName)
		})

		// Employee endpoints
		r.Route("/employees", func(r chi.Router) {
			r.With(s.authMiddleware.RequireAuth).Post("/invite", s.employeeHandler.InviteEmployee) // Create employee invitation (matches Python POST /employees)
			r.With(s.authMiddleware.RequireAuth).Get("/", s.employeeHandler.ListEmployees)
			r.With(s.authMiddleware.RequireAuth).Get("/me", s.employeeHandler.GetMe) // Must be before /{id} to avoid conflicts
			r.With(s.authMiddleware.RequireAuth).Get("/{id}", s.employeeHandler.GetEmployee)
			r.With(s.authMiddleware.RequireAuth).Put("/{id}", s.employeeHandler.UpdateEmployee)
			r.With(s.authMiddleware.RequireAuth).Delete("/{id}", s.employeeHandler.DeleteEmployee)

			// Employee invitation management endpoints
			r.Route("/invitations", func(r chi.Router) {
				r.Use(s.authMiddleware.RequireAuth)
				r.Get("/pending", s.employeeHandler.GetPendingInvitations)
				r.Delete("/{invitationId}", s.employeeHandler.DeleteInvitation)
				r.Post("/extend", s.employeeHandler.ExtendInvitation)
				r.Patch("/", s.employeeHandler.UpdateInvitation)
			})

			// Employee profile management endpoints
			r.Route("/me", func(r chi.Router) {
				r.Use(s.authMiddleware.RequireAuth)
				r.Patch("/", s.employeeHandler.UpdateMe)
				r.Post("/email/update", s.employeeHandler.UpdateEmail)
				r.Post("/email/verify", s.employeeHandler.VerifyEmailUpdate)
				r.Post("/telegram/connect", s.employeeHandler.ConnectTelegram)
				r.Get("/telegram/connect/{requestId}", s.employeeHandler.CheckTelegramConnection)
				r.Delete("/telegram", s.employeeHandler.DisconnectTelegram)
				r.Delete("/email", s.employeeHandler.DisconnectEmail)
			})
		})

		// Restaurant endpoints
		r.Route("/restaurants", func(r chi.Router) {
			// Any authenticated user can create a restaurant (bootstrap case) - they automatically become owner
			r.With(s.authMiddleware.RequireAuth).Post("/", s.restaurantHandler.CreateRestaurant)
			r.With(s.authMiddleware.RequireAuth).Get("/", s.restaurantHandler.ListRestaurants)

			// Special endpoints that need to be before /{id} patterns
			r.Get("/url-name-availability", s.restaurantHandler.CheckURLNameAvailability)

			// Dynamic restaurant identifier endpoints (handles both ObjectID and URL name)
			r.Get("/{urlNameOrRestId}", s.restaurantHandler.GetRestaurantByURLNameOrID)
			r.Get("/{urlNameOrRestId}/availability", s.restaurantHandler.GetRestaurantAvailability)

			// Restaurant management endpoints (require ObjectID)
			r.Put("/{id}", s.restaurantHandler.UpdateRestaurant)
			r.Patch("/{id}", s.restaurantHandler.UpdateRestaurant) // PATCH also supported for updates
			r.Delete("/{id}", s.restaurantHandler.DeleteRestaurant)
			r.Patch("/{restaurantId}/settings", s.restaurantHandler.UpdateRestaurantSettings)
			r.Post("/{restaurantId}/enable", s.restaurantHandler.EnableRestaurant)
			r.Post("/{restaurantId}/disable", s.restaurantHandler.DisableRestaurant)

			// Sub-URL management endpoints
			r.Post("/{restaurantId}/sub-url", s.restaurantHandler.CreateSubURL)
			r.Delete("/{restaurantId}/sub-url/{subUrlId}", s.restaurantHandler.DeleteSubURL)

			// Restaurant reservation management endpoints
			r.Patch("/{restaurantId}/reservations/mark_seen", s.restaurantHandler.MarkReservationsAsSeen)
			r.Get("/{restaurantId}/reservations/counts", s.restaurantHandler.GetReservationCounts)

			// Room endpoints for restaurants
			r.Route("/{restaurantId}/rooms", func(r chi.Router) {
				r.Get("/", s.roomHandler.GetRooms)
				r.Post("/", s.roomHandler.CreateRoom)
				r.Get("/{roomId}", s.roomHandler.GetRoom)
				r.Patch("/{roomId}", s.roomHandler.UpdateRoom)
				r.Delete("/{roomId}", s.roomHandler.DeleteRoom)
				r.Post("/{roomId}/enable", s.roomHandler.EnableRoom)
				r.Post("/{roomId}/disable", s.roomHandler.DisableRoom)

				// Element endpoints for rooms
				r.Post("/{roomId}/elements", s.roomHandler.SaveElements)
				r.Patch("/{roomId}/elements/{elementId}", s.roomHandler.UpdateElement)
				r.Patch("/{roomId}/elements/{elementId}/enable", s.roomHandler.EnableElement)
				r.Patch("/{roomId}/elements/{elementId}/disable", s.roomHandler.DisableElement)

				// Reservation endpoints for rooms
				r.Post("/{roomId}/reservations/by_employee", s.reservationHandler.CreateReservationByEmployee)
				r.Get("/{roomId}/reservations", s.reservationHandler.GetReservationsByRoom)
			})
		})

		// Individual reservation management endpoints
		r.Route("/reservations", func(r chi.Router) {
			r.Get("/{reservationId}", s.reservationHandler.GetReservation)
			r.Patch("/{reservationId}/by_admin", s.reservationHandler.UpdateReservationByAdmin)
			r.Patch("/{reservationId}/status", s.reservationHandler.UpdateReservationStatus)
			r.Post("/{reservationId}/cancel", s.reservationHandler.CancelReservation)
		})

		// All required endpoints implemented
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
	if _, err := w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
		s.logger.Error("Failed to write health check response", "error", err)
	}
}
