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
	sqlDB, err := database.GetSQLDB(db)
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}

	// Применяем миграции
	if err := database.MigrateDB(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Инициализируем подключение к Redis
	redisClient, err := database.NewRedisClient(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Инициализируем репозитории
	userRepo := pgRepo.NewUserRepo(db)
	quizRepo := pgRepo.NewQuizRepo(db)
	questionRepo := pgRepo.NewQuestionRepo(db)
	resultRepo := pgRepo.NewResultRepo(db)
	cacheRepo := redisRepo.NewCacheRepo(redisClient)

	// Инициализируем репозиторий для инвалидированных токенов
	invalidTokenRepo := pgRepo.NewInvalidTokenRepo(db)

	// Инициализируем репозиторий для refresh-токенов
	refreshTokenRepo := pgRepo.NewRefreshTokenRepo(sqlDB)

	// Создаем JWT сервис с поддержкой персистентного хранения инвалидированных токенов
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpirationHrs, invalidTokenRepo)

	// Инициализируем TokenService для управления access и refresh токенами
	tokenService := auth.NewTokenService(jwtService, refreshTokenRepo, userRepo)

	// Создаем контекст с отменой для корректного завершения работы горутин
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем фоновую задачу для очистки инвалидированных токенов
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		log.Println("Запуск механизма периодической очистки инвалидированных токенов (каждые 30 минут)")

		// Сразу выполняем первую очистку при запуске
		log.Println("Выполняю первичную очистку инвалидированных токенов...")
		if err := jwtService.CleanupInvalidatedUsers(); err != nil {
			log.Printf("Ошибка при первичной очистке инвалидированных токенов: %v", err)
		} else {
			log.Println("Первичная очистка инвалидированных токенов выполнена успешно")
		}

		for {
			select {
			case <-ticker.C:
				log.Println("Выполняю периодическую очистку инвалидированных токенов...")
				if err := jwtService.CleanupInvalidatedUsers(); err != nil {
					log.Printf("Ошибка при очистке инвалидированных токенов: %v", err)
				} else {
					log.Println("Очистка инвалидированных токенов выполнена успешно")
				}
			case <-ctx.Done():
				log.Println("Завершение работы горутины очистки инвалидированных токенов")
				return
			}
		}
	}()

	// Инициализируем токен-менеджер для улучшенного управления токенами и безопасности
	tokenManager := manager.NewTokenManager(jwtService, tokenService, refreshTokenRepo, userRepo)

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

	// Инициализируем WebSocket Hub и Manager
	wsConfig := ws.ShardedHubConfig{
		ShardCount:         cfg.WebSocket.Sharding.ShardCount,
		MaxClientsPerShard: cfg.WebSocket.Sharding.MaxClientsPerShard,
		ClusterConfig: ws.ClusterConfig{
			Enabled:          cfg.WebSocket.Cluster.Enabled,
			InstanceID:       cfg.WebSocket.Cluster.InstanceID,
			BroadcastChannel: cfg.WebSocket.Cluster.BroadcastChannel,
			DirectChannel:    cfg.WebSocket.Cluster.DirectChannel,
			MetricsChannel:   cfg.WebSocket.Cluster.MetricsChannel,
			MetricsInterval:  time.Duration(cfg.WebSocket.Cluster.MetricsInterval) * time.Second,
			Provider:         &ws.NoOpPubSub{}, // По умолчанию используем NoOp, можно заменить на Redis
		},
	}

	// Если кластеризация включена, настраиваем Redis PubSub
	if cfg.WebSocket.Cluster.Enabled {
		log.Println("Инициализация Redis PubSub для кластеризации WebSocket...")
		// Создаем конфигурацию Redis для PubSub
		redisPubSubConfig := ws.RedisConfig{
			Addresses:  []string{cfg.Redis.Addr},
			Password:   cfg.Redis.Password,
			DB:         cfg.Redis.DB,
			UseCluster: false,
			MaxRetries: 3,
		}

		// Создаем провайдер Redis PubSub
		redisPubSub, err := ws.NewRedisPubSub(redisPubSubConfig)
		if err != nil {
			log.Printf("Ошибка при инициализации Redis PubSub: %v", err)
			log.Println("Продолжаем с NoOpPubSub в одиночном режиме...")
		} else {
			log.Println("Redis PubSub успешно инициализирован")
			// Устанавливаем Redis PubSub как провайдера для кластера
			wsConfig.ClusterConfig.Provider = redisPubSub
		}
	}

	wsHub := ws.NewShardedHub(wsConfig)
	go wsHub.Run()
	wsManager := ws.NewManager(wsHub)

	// Инициализируем сервисы
	authService := service.NewAuthService(userRepo, jwtService, tokenService, refreshTokenRepo)
	// Устанавливаем TokenManager для сервисов
	authService.WithTokenManager(tokenManager)
	quizService := service.NewQuizService(quizRepo, questionRepo, cacheRepo)
	resultService := service.NewResultService(resultRepo, userRepo, quizRepo, questionRepo, cacheRepo)
	quizManager := service.NewQuizManager(quizRepo, questionRepo, resultRepo, resultService, cacheRepo, wsManager)

	// Инициализируем обработчики
	authHandler := handler.NewAuthHandler(authService, tokenManager, wsHub)
	quizHandler := handler.NewQuizHandler(quizService, resultService, quizManager)
	wsHandler := handler.NewWSHandler(wsHub, wsManager, quizManager, jwtService)

	// Инициализируем middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService, tokenService)
	// Добавляем TokenManager к middleware
	authMiddleware.WithTokenManager(tokenManager)

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
			quizzes.GET("/:id", quizHandler.GetQuiz)
			quizzes.GET("/:id/with-questions", quizHandler.GetQuizWithQuestions)
			quizzes.GET("/:id/results", quizHandler.GetQuizResults)

			// Маршруты для аутентифицированных пользователей
			authedQuizzes := quizzes.Group("/")
			authedQuizzes.Use(authMiddleware.RequireAuth())
			{
				authedQuizzes.GET("/:id/my-result", quizHandler.GetUserQuizResult)
			}

			// Маршруты для администраторов
			adminQuizzes := quizzes.Group("/")
			adminQuizzes.Use(authMiddleware.RequireAuth(), authMiddleware.AdminOnly())
			{
				adminQuizzes.POST("", quizHandler.CreateQuiz)
				adminQuizzes.POST("/:id/questions", quizHandler.AddQuestions)
				adminQuizzes.PUT("/:id/schedule", quizHandler.ScheduleQuiz)
				adminQuizzes.PUT("/:id/cancel", quizHandler.CancelQuiz)
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

	// Создаем контекст с таймаутом для graceful shutdown сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
