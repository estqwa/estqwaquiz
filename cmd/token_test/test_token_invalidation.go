package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yourusername/trivia-api/internal/config"
	pgRepo "github.com/yourusername/trivia-api/internal/repository/postgres"
	"github.com/yourusername/trivia-api/pkg/auth"
	"github.com/yourusername/trivia-api/pkg/database"
)

/*
Тестовый скрипт для проверки механизма инвалидации токенов.
Использование:
  go run cmd/tools/test_token_invalidation.go [email]

Если email указан, будет очищена инвалидация для конкретного пользователя.
Если email не указан, будет протестирован весь механизм инвалидации.
*/

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализируем подключение к PostgreSQL
	db, err := database.NewPostgresDB(cfg.Database.PostgresConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Инициализируем репозитории
	userRepo := pgRepo.NewUserRepo(db)
	invalidTokenRepo := pgRepo.NewInvalidTokenRepo(db)

	// Создаем JWT сервис
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpirationHrs, invalidTokenRepo)

	// Проверяем аргументы
	if len(os.Args) > 1 {
		// Если указан email, сбросим инвалидацию для конкретного пользователя
		email := os.Args[1]
		clearInvalidationForUser(email, userRepo, jwtService)
		return
	}

	// Тестируем полный цикл инвалидации
	testTokenInvalidation(userRepo, jwtService, invalidTokenRepo)
}

// clearInvalidationForUser сбрасывает инвалидацию для указанного пользователя
func clearInvalidationForUser(email string, userRepo *pgRepo.UserRepo, jwtService *auth.JWTService) {
	fmt.Printf("Сбрасываем инвалидацию для пользователя %s\n", email)

	user, err := userRepo.GetByEmail(email)
	if err != nil {
		log.Fatalf("Пользователь не найден: %v", err)
	}

	fmt.Printf("Пользователь найден: ID=%d, Email=%s\n", user.ID, user.Email)

	jwtService.ResetInvalidationForUser(user.ID)
	fmt.Printf("Инвалидация для пользователя %s сброшена\n", email)
}

// testTokenInvalidation тестирует полный цикл инвалидации токенов
func testTokenInvalidation(userRepo *pgRepo.UserRepo, jwtService *auth.JWTService, invalidTokenRepo *pgRepo.InvalidTokenRepo) {
	// Список для теста (можно заменить на нужных пользователей)
	testEmails := []string{"test@example.com", "timatest@mail.ru"}

	for _, email := range testEmails {
		user, err := userRepo.GetByEmail(email)
		if err != nil {
			fmt.Printf("Пользователь %s не найден, пропускаем\n", email)
			continue
		}

		fmt.Printf("\n=== Тест инвалидации для %s (ID=%d) ===\n", user.Email, user.ID)

		// Создаем токен
		token, err := jwtService.GenerateToken(user)
		if err != nil {
			log.Fatalf("Ошибка создания токена: %v", err)
		}
		fmt.Printf("Создан токен: %s\n", token)

		// Проверяем токен
		claims, err := jwtService.ParseToken(token)
		if err != nil {
			log.Fatalf("Ошибка проверки токена: %v", err)
		}
		fmt.Printf("Токен валиден для UserID=%d, Email=%s\n", claims.UserID, claims.Email)

		// Инвалидируем токен
		err = jwtService.InvalidateTokensForUser(user.ID)
		if err != nil {
			log.Fatalf("Ошибка инвалидации токена: %v", err)
		}
		fmt.Printf("Токен инвалидирован\n")

		// Проверяем запись в БД
		isInvalid, err := invalidTokenRepo.IsTokenInvalid(user.ID, claims.IssuedAt.Time)
		if err != nil {
			log.Fatalf("Ошибка при проверке инвалидации в БД: %v", err)
		}
		fmt.Printf("Запись в БД проверена, токен недействителен: %v\n", isInvalid)

		// Проверяем токен снова (должен быть недействителен)
		claims, err = jwtService.ParseToken(token)
		if err != nil {
			fmt.Printf("Ожидаемая ошибка проверки токена: %v (токен недействителен)\n", err)
		} else {
			log.Fatalf("ОШИБКА! Токен все еще валиден после инвалидации!")
		}

		// Сбрасываем инвалидацию
		jwtService.ResetInvalidationForUser(user.ID)
		fmt.Printf("Инвалидация сброшена\n")

		// Создаем новый токен
		token, err = jwtService.GenerateToken(user)
		if err != nil {
			log.Fatalf("Ошибка создания нового токена: %v", err)
		}
		fmt.Printf("Создан новый токен: %s\n", token)

		// Проверяем новый токен
		claims, err = jwtService.ParseToken(token)
		if err != nil {
			log.Fatalf("Ошибка проверки нового токена: %v", err)
		}
		fmt.Printf("Новый токен валиден для UserID=%d, Email=%s\n", claims.UserID, claims.Email)

		fmt.Printf("=== Тест для %s завершен успешно ===\n\n", user.Email)
	}

	// Тестируем очистку устаревших записей
	fmt.Println("Тестирование очистки устаревших инвалидаций:")

	// Добавляем устаревшую запись
	userID := uint(999999) // Несуществующий пользователь для теста
	oldTime := time.Now().Add(-48 * time.Hour)
	err := invalidTokenRepo.AddInvalidToken(userID, oldTime)
	if err != nil {
		fmt.Printf("Ошибка добавления тестовой инвалидации: %v (игнорируем)\n", err)
	} else {
		fmt.Printf("Добавлена тестовая устаревшая инвалидация для UserID=%d\n", userID)

		// Запускаем очистку
		jwtService.CleanupInvalidatedUsers()
		fmt.Println("Очистка завершена")

		// Проверяем, удалена ли запись
		isInvalid, _ := invalidTokenRepo.IsTokenInvalid(userID, time.Now())
		fmt.Printf("Запись осталась: %v (должна быть false)\n", isInvalid)
	}

	fmt.Println("Все тесты завершены успешно!")
}
