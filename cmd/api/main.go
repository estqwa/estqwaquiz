package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/yourusername/trivia-api/internal/config"
	"github.com/yourusername/trivia-api/internal/handler"
	"github.com/yourusername/trivia-api/internal/middleware"
	pgRepo "github.com/yourusername/trivia-api/internal/repository/postgres"
	redisRepo "github.com/yourusername/trivia-api/internal/repository/redis"
	"github.com/yourusername/trivia-api/internal/service"
	ws "github.com/yourusername/trivia-api/internal/websocket"
	"github.com/yourusername/trivia-api/pkg/auth"
	"github.com/yourusername/trivia-api/pkg/auth/manager"
	"github.com/yourusername/trivia-api/pkg/database"
)

func main() {
	// Загружаем конфигурацию
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}
	log.Printf("Загрузка конфигурации из %s", configPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализируем подключение к PostgreSQL
	db, err := database.NewPostgresDB(cfg.Database.PostgresConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Получаем sql.DB для репозитория refresh-токенов
	// sqlDB, err := database.GetSQLDB(db) // REMOVED
	// if err != nil {                     // REMOVED
	// 	log.Fatalf("Failed to get sql.DB: %v", err) // REMOVED
	// }                                     // REMOVED

	// Применяем миграции
	if err := database.MigrateDB(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Инициализируем подключение к Redis с использованием унифицированной конфигурации
	redisClient, err := database.NewUniversalRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Инициализируем репозитории
	userRepo := pgRepo.NewUserRepo(db)
	quizRepo := pgRepo.NewQuizRepo(db)
	questionRepo := pgRepo.NewQuestionRepo(db)
	resultRepo := pgRepo.NewResultRepo(db)
	cacheRepo := redisRepo.NewCacheRepo(redisClient)

	// Инициализируем репозиторий для инвалидированных токенов
	invalidTokenRepo := pgRepo.NewInvalidTokenRepo(db)

	// Инициализируем репозиторий для refresh-токенов
	refreshTokenRepo := pgRepo.NewRefreshTokenRepo(db)

	// Создаем JWT сервис с поддержкой персистентного хранения инвалидированных токенов
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpirationHrs, invalidTokenRepo, cfg.JWT.WSTicketExpirySec, cfg.JWT.CleanupInterval)

	// Создаем TokenManager
	tokenManager := manager.NewTokenManager(jwtService, refreshTokenRepo, userRepo)
	tokenManager.SetAccessTokenExpiry(time.Duration(cfg.JWT.ExpirationHrs) * time.Hour)          // Используем значение из конфига
	tokenManager.SetRefreshTokenExpiry(time.Duration(cfg.Auth.RefreshTokenLifetime) * time.Hour) // Используем значение из конфига
	tokenManager.SetMaxRefreshTokensPerUser(cfg.Auth.SessionLimit)                               // Используем значение из конфига
	tokenManager.SetProductionMode(gin.Mode() == gin.ReleaseMode)                                // Устанавливаем режим для Secure кук

	// Передаем TokenManager в AuthService
	authService := service.NewAuthService(userRepo, jwtService, tokenManager, refreshTokenRepo, invalidTokenRepo)

	// Создаем контекст с отменой для корректного завершения работы горутин
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем фоновую задачу для очистки истекших CSRF токенов и других ресурсов
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		log.Println("Запуск механизма периодической очистки CSRF токенов (каждый час)")

		for {
			select {
			case <-ticker.C:
				log.Println("Выполняю периодическую очистку CSRF токенов и истекших refresh-токенов...")
				if err := tokenManager.CleanupExpiredTokens(); err != nil {
					log.Printf("Ошибка при очистке токенов: %v", err)
				} else {
					log.Println("Очистка токенов выполнена успешно")
				}
			case <-ctx.Done():
				log.Println("Завершение работы горутины очистки токенов")
				return
			}
		}
	}()

	// --- Инициализация WebSocket --- //
	var wsHub ws.HubInterface
	var pubSubProvider ws.PubSubProvider = &ws.NoOpPubSub{} // Провайдер по умолчанию

	// Создаем PubSubProvider только если кластеризация включена
	if cfg.WebSocket.Cluster.Enabled {
		log.Println("Инициализация Redis PubSub для кластеризации WebSocket...")
		// Создаем КЛИЕНТ Redis PubSub с использованием той же универсальной функции
		redisPubSubClient, err := database.NewUniversalRedisClient(cfg.Redis)
		if err != nil {
			log.Printf("Ошибка при инициализации Redis клиента для PubSub: %v. Кластеризация WS будет неактивна.", err)
			// Используем NoOpPubSub если не удалось подключиться к Redis
			pubSubProvider = &ws.NoOpPubSub{}
		} else {
			// Создаем провайдер Redis PubSub, передавая ему созданный клиент
			redisProvider, err := ws.NewRedisPubSub(redisPubSubClient)
			if err != nil {
				log.Printf("Ошибка при создании Redis PubSub провайдера: %v. Кластеризация WS будет неактивна.", err)
				redisPubSubClient.Close() // Закрываем созданный клиент, так как он не будет использоваться
				pubSubProvider = &ws.NoOpPubSub{}
			} else {
				log.Println("Redis PubSub провайдер успешно инициализирован")
				pubSubProvider = redisProvider
			}
		}
	}

	if cfg.WebSocket.Sharding.Enabled {
		log.Println("WebSocket: включено шардирование")
		// Передаем конфигурацию WebSocket и PubSubProvider в ShardedHub
		shardedHub := ws.NewShardedHub(cfg.WebSocket, pubSubProvider)
		go shardedHub.Run() // Запускаем обработчик шардов
		wsHub = shardedHub
	} else {
		log.Println("WebSocket: используется один хаб")
		// Для простого Hub не требуется сложная конфигурация или PubSub
		hub := ws.NewHub()
		go hub.Run()
		wsHub = hub
	}

	wsManager := ws.NewManager(wsHub)

	// Инициализируем сервисы
	quizService := service.NewQuizService(quizRepo, questionRepo, cacheRepo)
	resultService := service.NewResultService(resultRepo, userRepo, quizRepo, questionRepo, cacheRepo, db, wsManager)
	quizManager := service.NewQuizManager(quizRepo, questionRepo, resultRepo, resultService, cacheRepo, wsManager, db)

	// Инициализируем обработчики
	authHandler := handler.NewAuthHandler(authService, tokenManager, wsHub)
	quizHandler := handler.NewQuizHandler(quizService, resultService, quizManager)
	wsHandler := handler.NewWSHandler(wsHub, wsManager, quizManager, jwtService)

	// Инициализируем middleware
	authMiddleware := middleware.NewAuthMiddlewareWithManager(jwtService, tokenManager)

	// Инициализируем роутер Gin
	router := gin.Default()

	// Настройка CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:8000", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Статические файлы для админ-панели
	router.StaticFS("/admin", http.Dir("./static/admin"))

	// Настраиваем маршруты API
	api := router.Group("/api")
	{
		// Аутентификация
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/check-refresh", authHandler.CheckRefreshToken)
			auth.POST("/token-info", authHandler.GetTokenInfo)

			// Маршруты, требующие аутентификации
			authedAuth := auth.Group("/")
			authedAuth.Use(authMiddleware.RequireAuth())
			{
				authedAuth.POST("/logout", authHandler.Logout)
				authedAuth.POST("/logout-all", authHandler.LogoutAllDevices)
				authedAuth.GET("/sessions", authHandler.GetActiveSessions)
				authedAuth.POST("/revoke-session", authHandler.RevokeSession)
				authedAuth.POST("/change-password", authHandler.ChangePassword)
				authedAuth.POST("/ws-ticket", authHandler.GenerateWsTicket)
			}

			// Маршрут для сброса инвалидаций токенов (только для администраторов)
			adminAuth := auth.Group("/admin")
			adminAuth.Use(authMiddleware.RequireAuth(), authMiddleware.AdminOnly())
			{
				adminAuth.POST("/reset-auth", authHandler.ResetAuth)
				adminAuth.POST("/debug-token", authHandler.DebugToken)
				adminAuth.POST("/reset-password", authHandler.AdminResetPassword)
			}
		}

		// Пользователи
		users := api.Group("/users")
		users.Use(authMiddleware.RequireAuth())
		{
			users.GET("/me", authHandler.GetMe)
			users.PUT("/me", authHandler.UpdateProfile)
		}

		// Викторины
		quizzes := api.Group("/quizzes")
		{
			quizzes.GET("", quizHandler.ListQuizzes)
			quizzes.GET("/active", quizHandler.GetActiveQuiz)
			quizzes.GET("/scheduled", quizHandler.GetScheduledQuizzes)

			// Группа маршрутов, требующих quizID
			quizWithID := quizzes.Group("/:id")
			quizWithID.Use(middleware.ExtractUintParam("id", "quizID")) // Применяем middleware
			{
				quizWithID.GET("", quizHandler.GetQuiz)
				quizWithID.GET("/with-questions", quizHandler.GetQuizWithQuestions)
				quizWithID.GET("/results", quizHandler.GetQuizResults)

				// Маршруты для аутентифицированных пользователей
				authedQuizzes := quizWithID.Group("") // Наследует middleware
				authedQuizzes.Use(authMiddleware.RequireAuth())
				{
					authedQuizzes.GET("/my-result", quizHandler.GetUserQuizResult)
				}

				// Маршруты для администраторов
				adminQuizzes := quizWithID.Group("") // Наследует middleware
				adminQuizzes.Use(authMiddleware.RequireAuth(), authMiddleware.AdminOnly())
				{
					adminQuizzes.POST("/questions", quizHandler.AddQuestions)
					adminQuizzes.PUT("/schedule", quizHandler.ScheduleQuiz)
					adminQuizzes.PUT("/cancel", quizHandler.CancelQuiz)
				}
			}

			// Маршрут создания викторины (не требует ID)
			adminCreateQuiz := quizzes.Group("")
			adminCreateQuiz.Use(authMiddleware.RequireAuth(), authMiddleware.AdminOnly())
			{
				adminCreateQuiz.POST("", quizHandler.CreateQuiz)
			}
		}
	}

	// WebSocket маршрут
	router.GET("/ws", wsHandler.HandleConnection)

	// Запланированные викторины
	// После перезапуска сервера нужно заново запланировать активные викторины
	go func() {
		scheduledQuizzes, err := quizService.GetScheduledQuizzes()
		if err != nil {
			log.Printf("Failed to get scheduled quizzes: %v", err)
			return
		}

		for _, quiz := range scheduledQuizzes {
			if err := quizManager.ScheduleQuiz(quiz.ID, quiz.ScheduledTime); err != nil {
				log.Printf("Failed to reschedule quiz %d: %v", quiz.ID, err)
			}
		}
	}()

	// Настраиваем HTTP сервер
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Запускаем сервер в горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.Server.Port)

	// В обработчике сигналов остановки
	// После получения сигнала SIGINT или SIGTERM вызываем cancel() для завершения горутин
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Отправляем сигнал завершения для всех горутин
	cancel()

	// Закрываем PubSubProvider, если он был создан
	if pubSubProvider != nil {
		if err := pubSubProvider.Close(); err != nil {
			log.Printf("Error closing PubSub provider: %v", err)
		}
	}

	// Создаем контекст с таймаутом для graceful shutdown сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
