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
	
	s.setupMiddleware(r)
	s.setupPublicRoutes(r)
	s.setupAPIRoutes(r)
	
	s.router = r
}

// setupMiddleware configures all middleware for the router.
func (s *Server) setupMiddleware(r *chi.Mux) {
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
}

// setupPublicRoutes configures public routes that don't require authentication.
func (s *Server) setupPublicRoutes(r *chi.Mux) {
	// Health check
	r.Get("/health", s.healthCheck)

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Prometheus metrics endpoint
	r.Get("/metrics", s.metricsHandler.GetMetrics)
}

// setupAPIRoutes configures all API routes under /api/admin.
func (s *Server) setupAPIRoutes(r *chi.Mux) {
	r.Route("/api/admin", func(r chi.Router) {
		// Swagger documentation under /api/admin/docs
		r.Get("/docs/*", httpSwagger.Handler(
			httpSwagger.URL("/api/admin/docs/doc.json"),
		))
		
		s.setupDevRoutes(r)
		s.setupAuthRoutes(r)
		s.setupCityRoutes(r)
		s.setupEmployeeRoutes(r)
		s.setupRestaurantRoutes(r)
		s.setupReservationRoutes(r)
	})
}

// setupDevRoutes configures development endpoints.
func (s *Server) setupDevRoutes(r chi.Router) {
	r.Route("/dev", func(r chi.Router) {
		r.Get("/ping", s.devHandler.Ping)
	})
}

// setupAuthRoutes configures authentication endpoints.
func (s *Server) setupAuthRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", s.authHandler.Login)
		r.Post("/register", s.authHandler.Register)
		r.Post("/verify", s.authHandler.Verify)
		r.Get("/register/employee", s.authHandler.RegisterEmployee)
		r.Get("/tg", s.authHandler.AuthorizeTelegram)
		r.Get("/tg/{requestID}", s.authHandler.VerifyTelegram)
	})
}

// setupCityRoutes configures city endpoints.
func (s *Server) setupCityRoutes(r chi.Router) {
	r.Route("/cities", func(r chi.Router) {
		r.Get("/", s.cityHandler.GetCities)
		r.Get("/{name}", s.cityHandler.GetCityByName)
	})
}

// setupEmployeeRoutes configures employee endpoints.
func (s *Server) setupEmployeeRoutes(r chi.Router) {
	r.Route("/employees", func(r chi.Router) {
		// Apply auth middleware to all employee routes
		r.Use(s.authMiddleware.RequireAuth)
		
		// Main employee operations
		r.Post("/invite", s.employeeHandler.InviteEmployee)
		r.Get("/", s.employeeHandler.ListEmployees)
		
		// Profile routes must be before /{id} to avoid conflicts
		s.setupEmployeeProfileRoutes(r)
		
		// Dynamic ID routes
		r.Get("/{id}", s.employeeHandler.GetEmployee)
		r.Put("/{id}", s.employeeHandler.UpdateEmployee)
		r.Delete("/{id}", s.employeeHandler.DeleteEmployee)

		s.setupEmployeeInvitationRoutes(r)
	})
}

// setupEmployeeInvitationRoutes configures employee invitation management endpoints.
func (s *Server) setupEmployeeInvitationRoutes(r chi.Router) {
	r.Route("/invitations", func(r chi.Router) {
		r.Get("/pending", s.employeeHandler.GetPendingInvitations)
		r.Delete("/{invitationId}", s.employeeHandler.DeleteInvitation)
		r.Post("/extend", s.employeeHandler.ExtendInvitation)
		r.Patch("/", s.employeeHandler.UpdateInvitation)
	})
}

// setupEmployeeProfileRoutes configures employee profile management endpoints.
func (s *Server) setupEmployeeProfileRoutes(r chi.Router) {
	r.Route("/me", func(r chi.Router) {
		r.Get("/", s.employeeHandler.GetMe) // GET /employees/me
		r.Patch("/", s.employeeHandler.UpdateMe)
		r.Post("/email/update", s.employeeHandler.UpdateEmail)
		r.Post("/email/verify", s.employeeHandler.VerifyEmailUpdate)
		r.Post("/telegram/connect", s.employeeHandler.ConnectTelegram)
		r.Get("/telegram/connect/{requestId}", s.employeeHandler.CheckTelegramConnection)
		r.Delete("/telegram", s.employeeHandler.DisconnectTelegram)
		r.Delete("/email", s.employeeHandler.DisconnectEmail)
	})
}

// setupRestaurantRoutes configures restaurant endpoints.
func (s *Server) setupRestaurantRoutes(r chi.Router) {
	r.Route("/restaurants", func(r chi.Router) {
		// Authenticated restaurant operations
		r.With(s.authMiddleware.RequireAuth).Post("/", s.restaurantHandler.CreateRestaurant)
		r.With(s.authMiddleware.RequireAuth).Get("/", s.restaurantHandler.ListRestaurants)

		// Public restaurant operations
		r.Get("/url-name-availability", s.restaurantHandler.CheckURLNameAvailability)
		r.Get("/{urlNameOrRestId}", s.restaurantHandler.GetRestaurantByURLNameOrID)
		r.Get("/{urlNameOrRestId}/availability", s.restaurantHandler.GetRestaurantAvailability)

		// Restaurant management (requires auth)
		r.With(s.authMiddleware.RequireAuth).Put("/{id}", s.restaurantHandler.UpdateRestaurant)
		r.With(s.authMiddleware.RequireAuth).Patch("/{id}", s.restaurantHandler.UpdateRestaurant)
		r.With(s.authMiddleware.RequireAuth).Delete("/{id}", s.restaurantHandler.DeleteRestaurant)
		r.With(s.authMiddleware.RequireAuth).Patch("/{restaurantId}/settings", s.restaurantHandler.UpdateRestaurantSettings)
		r.With(s.authMiddleware.RequireAuth).Post("/{restaurantId}/enable", s.restaurantHandler.EnableRestaurant)
		r.With(s.authMiddleware.RequireAuth).Post("/{restaurantId}/disable", s.restaurantHandler.DisableRestaurant)

		s.setupRestaurantSubURLRoutes(r)
		s.setupRestaurantReservationRoutes(r)
		s.setupRoomRoutes(r)
	})
}

// setupRestaurantSubURLRoutes configures sub-URL management endpoints.
func (s *Server) setupRestaurantSubURLRoutes(r chi.Router) {
	r.With(s.authMiddleware.RequireAuth).Post("/{restaurantId}/sub-url", s.restaurantHandler.CreateSubURL)
	r.With(s.authMiddleware.RequireAuth).Delete("/{restaurantId}/sub-url/{subUrlId}", s.restaurantHandler.DeleteSubURL)
}

// setupRestaurantReservationRoutes configures restaurant-level reservation endpoints.
func (s *Server) setupRestaurantReservationRoutes(r chi.Router) {
	r.With(s.authMiddleware.RequireAuth).Patch("/{restaurantId}/reservations/mark_seen", s.restaurantHandler.MarkReservationsAsSeen)
	r.With(s.authMiddleware.RequireAuth).Get("/{restaurantId}/reservations/counts", s.restaurantHandler.GetReservationCounts)
}

// setupRoomRoutes configures room endpoints for restaurants.
func (s *Server) setupRoomRoutes(r chi.Router) {
	r.Route("/{restaurantId}/rooms", func(r chi.Router) {
		r.Use(s.authMiddleware.RequireAuth)
		
		// Room operations
		r.Get("/", s.roomHandler.GetRooms)
		r.Post("/", s.roomHandler.CreateRoom)
		r.Get("/{roomId}", s.roomHandler.GetRoom)
		r.Patch("/{roomId}", s.roomHandler.UpdateRoom)
		r.Delete("/{roomId}", s.roomHandler.DeleteRoom)
		r.Post("/{roomId}/enable", s.roomHandler.EnableRoom)
		r.Post("/{roomId}/disable", s.roomHandler.DisableRoom)

		s.setupRoomElementRoutes(r)
		s.setupRoomReservationRoutes(r)
	})
}

// setupRoomElementRoutes configures element endpoints for rooms.
func (s *Server) setupRoomElementRoutes(r chi.Router) {
	r.Post("/{roomId}/elements", s.roomHandler.SaveElements)
	r.Patch("/{roomId}/elements/{elementId}", s.roomHandler.UpdateElement)
	r.Patch("/{roomId}/elements/{elementId}/enable", s.roomHandler.EnableElement)
	r.Patch("/{roomId}/elements/{elementId}/disable", s.roomHandler.DisableElement)
}

// setupRoomReservationRoutes configures reservation endpoints for rooms.
func (s *Server) setupRoomReservationRoutes(r chi.Router) {
	r.Post("/{roomId}/reservations/by_employee", s.reservationHandler.CreateReservationByEmployee)
	r.Get("/{roomId}/reservations", s.reservationHandler.GetReservationsByRoom)
}

// setupReservationRoutes configures individual reservation management endpoints.
func (s *Server) setupReservationRoutes(r chi.Router) {
	r.Route("/reservations", func(r chi.Router) {
		r.Use(s.authMiddleware.RequireAuth)
		
		r.Get("/{reservationId}", s.reservationHandler.GetReservation)
		r.Patch("/{reservationId}/by_admin", s.reservationHandler.UpdateReservationByAdmin)
		r.Patch("/{reservationId}/status", s.reservationHandler.UpdateReservationStatus)
		r.Post("/{reservationId}/cancel", s.reservationHandler.CancelReservation)
	})
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
