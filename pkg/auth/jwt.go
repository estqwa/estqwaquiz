package auth

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
)

// JWTCustomClaims содержит пользовательские поля для токена
type JWTCustomClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// JWTService предоставляет методы для работы с JWT
type JWTService struct {
	secretKey     string
	expirationHrs int
	// Черный список для инвалидированных пользователей (in-memory)
	invalidatedUsers map[uint]time.Time
	// Мьютекс для безопасной работы с картой в многопоточной среде
	mu sync.RWMutex
	// Репозиторий для персистентного хранения инвалидированных токенов
	invalidTokenRepo repository.InvalidTokenRepository
}

// NewJWTService создает новый сервис JWT
func NewJWTService(secretKey string, expirationHrs int, invalidTokenRepo repository.InvalidTokenRepository) *JWTService {
	service := &JWTService{
		secretKey:        secretKey,
		expirationHrs:    expirationHrs,
		invalidatedUsers: make(map[uint]time.Time),
		invalidTokenRepo: invalidTokenRepo,
	}

	// Загружаем инвалидированные токены из БД при создании сервиса
	service.loadInvalidatedTokensFromDB()

	return service
}

// loadInvalidatedTokensFromDB загружает информацию об инвалидированных токенах из БД
func (s *JWTService) loadInvalidatedTokensFromDB() {
	// Если репозиторий не инициализирован, выходим
	if s.invalidTokenRepo == nil {
		log.Println("JWT: Repository not initialized, skipping DB load")
		return
	}

	tokens, err := s.invalidTokenRepo.GetAllInvalidTokens()
	if err != nil {
		log.Printf("JWT: Error loading invalidated tokens from DB: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, token := range tokens {
		s.invalidatedUsers[token.UserID] = token.InvalidationTime
	}

	log.Printf("JWT: Loaded %d invalidated tokens from database", len(tokens))
}

// GenerateToken создает новый JWT токен для пользователя
func (s *JWTService) GenerateToken(user *entity.User) (string, error) {
	claims := &JWTCustomClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(s.expirationHrs))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		log.Printf("[JWT] Ошибка генерации токена для пользователя ID=%d: %v", user.ID, err)
		return "", err
	}

	log.Printf("[JWT] Токен успешно сгенерирован для пользователя ID=%d, выдан в %v, истекает через %d часов",
		user.ID, claims.IssuedAt, s.expirationHrs)
	return tokenString, nil
}

// ParseToken проверяет и расшифровывает JWT токен
func (s *JWTService) ParseToken(tokenString string) (*JWTCustomClaims, error) {
	claims := &JWTCustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("[JWT] Неожиданный метод подписи: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		// Более подробное логирование ошибок JWT
		if ve, ok := err.(*jwt.ValidationError); ok {
			switch {
			case ve.Errors&jwt.ValidationErrorMalformed != 0:
				log.Printf("[JWT] Ошибка: Токен имеет неверный формат")
				return nil, errors.New("token is malformed")
			case ve.Errors&jwt.ValidationErrorExpired != 0:
				log.Printf("[JWT] Ошибка: Токен истек срок действия для пользователя ID=%d", claims.UserID)
				return nil, errors.New("token is expired")
			case ve.Errors&jwt.ValidationErrorNotValidYet != 0:
				log.Printf("[JWT] Ошибка: Токен еще не действителен")
				return nil, errors.New("token not valid yet")
			case ve.Errors&jwt.ValidationErrorSignatureInvalid != 0:
				log.Printf("[JWT] Ошибка: Неверная подпись токена")
				return nil, errors.New("signature is invalid")
			default:
				log.Printf("[JWT] Ошибка при разборе токена: %v", err)
				return nil, errors.New("token validation failed")
			}
		} else {
			log.Printf("[JWT] Ошибка при разборе токена: %v", err)
			return nil, err
		}
	}

	if !token.Valid {
		log.Printf("[JWT] Токен недействителен")
		return nil, errors.New("invalid token")
	}

	// Проверяем, является ли токен WS-тикетом
	if purpose, ok := token.Header["purpose"]; ok && purpose == "websocket_auth" {
		log.Printf("[JWT] Проверка WS-тикета для пользователя ID=%d", claims.UserID)
		// Для WS-тикетов пропускаем проверку инвалидации
		return claims, nil
	}

	// Проверка на инвалидацию токена (только для обычных токенов, не для WS-тикетов)
	isInvalidInDB := false
	if s.invalidTokenRepo != nil {
		var dbErr error
		isInvalidInDB, dbErr = s.invalidTokenRepo.IsTokenInvalid(claims.UserID, claims.IssuedAt.Time)
		if dbErr != nil {
			log.Printf("[JWT] Ошибка при проверке инвалидации токена в БД: %v", dbErr)
		}
	}

	isInvalidInMem := false
	if claims.UserID > 0 {
		s.mu.RLock()
		invalidationTime, exists := s.invalidatedUsers[claims.UserID]
		s.mu.RUnlock()

		if exists && !claims.IssuedAt.Time.After(invalidationTime) {
			isInvalidInMem = true
			log.Printf("[JWT] Время выдачи токена: %v, время инвалидации: %v",
				claims.IssuedAt.Time, invalidationTime)
		}
	}

	if isInvalidInDB || isInvalidInMem {
		if isInvalidInDB {
			log.Printf("[JWT] Токен инвалидирован в БД для пользователя ID=%d, выдан в %v",
				claims.UserID, claims.IssuedAt.Time)
		}
		if isInvalidInMem {
			log.Printf("[JWT] Токен инвалидирован в памяти для пользователя ID=%d, выдан в %v",
				claims.UserID, claims.IssuedAt.Time)
		}
		return nil, errors.New("token has been invalidated")
	}

	log.Printf("[JWT] Токен успешно проверен для пользователя ID=%d, Email=%s, выдан: %v",
		claims.UserID, claims.Email, claims.IssuedAt.Time)
	return claims, nil
}

// InvalidateTokensForUser добавляет пользователя в черный список,
// делая все ранее выданные токены недействительными
func (s *JWTService) InvalidateTokensForUser(userID uint) error {
	// Инвалидация в памяти
	s.mu.Lock()
	s.invalidatedUsers[userID] = time.Now()
	s.mu.Unlock()

	// Инвалидация в БД
	if s.invalidTokenRepo != nil {
		err := s.invalidTokenRepo.AddInvalidToken(userID, time.Now())
		if err != nil {
			log.Printf("[JWT] Ошибка при добавлении записи инвалидации в БД для пользователя ID=%d: %v",
				userID, err)
			return err
		}
	}

	log.Printf("[JWT] Токены инвалидированы для пользователя ID=%d в %v", userID, time.Now())
	return nil
}

// ResetInvalidationForUser удаляет пользователя из черного списка,
// разрешая использование существующих токенов
func (s *JWTService) ResetInvalidationForUser(userID uint) {
	if userID == 0 {
		log.Printf("JWT: Попытка сброса инвалидации для некорректного UserID: %d", userID)
		return
	}

	s.mu.Lock()
	_, exists := s.invalidatedUsers[userID]
	if exists {
		delete(s.invalidatedUsers, userID)
		log.Printf("JWT: Reset invalidation for UserID: %d", userID)
	} else {
		log.Printf("JWT: UserID: %d was not in the invalidation list", userID)
	}
	s.mu.Unlock()

	// Удаляем также из БД, если репозиторий инициализирован
	if s.invalidTokenRepo != nil {
		err := s.invalidTokenRepo.RemoveInvalidToken(userID)
		if err != nil {
			log.Printf("JWT: Error removing invalidation from DB for UserID: %d: %v", userID, err)
			// Продолжаем выполнение
		}
	}

	log.Printf("JWT: Инвалидация сброшена для пользователя ID=%d, токены снова действительны", userID)
}

// CleanupInvalidatedUsers удаляет записи об инвалидированных пользователях через 24 часа
// Для предотвращения бесконечного роста карты
func (s *JWTService) CleanupInvalidatedUsers() error {
	// Увеличиваем период хранения инвалидированных токенов с 24 до 48 часов
	// для большей безопасности
	cutoffTime := time.Now().Add(-48 * time.Hour)

	// Создаем список пользователей для удаления, чтобы не изменять
	// карту во время итерации
	var usersToRemove []uint

	// Очищаем в памяти
	s.mu.Lock()
	beforeCount := len(s.invalidatedUsers)
	for userID, invalidationTime := range s.invalidatedUsers {
		if invalidationTime.Before(cutoffTime) {
			usersToRemove = append(usersToRemove, userID)
		}
	}

	// Удаляем из карты
	for _, userID := range usersToRemove {
		delete(s.invalidatedUsers, userID)
	}
	s.mu.Unlock()

	// Если база данных доступна, очищаем также в ней
	if s.invalidTokenRepo != nil {
		if err := s.invalidTokenRepo.CleanupOldInvalidTokens(cutoffTime); err != nil {
			log.Printf("JWT: Ошибка при очистке инвалидированных токенов в БД: %v", err)
			return fmt.Errorf("ошибка очистки инвалидированных токенов в БД: %w", err)
		}
		log.Printf("JWT: Очистка устаревших записей об инвалидации в БД выполнена успешно")
	} else {
		log.Printf("JWT: Репозиторий токенов не доступен, очистка в БД не выполнена")
	}

	log.Printf("JWT: Удалено %d устаревших записей об инвалидации из памяти", len(usersToRemove))
	log.Printf("JWT: Карта инвалидированных токенов: было %d, стало %d", beforeCount, len(s.invalidatedUsers))
	return nil
}

// DebugToken анализирует JWT токен без проверки подписи
// для диагностических целей
func (s *JWTService) DebugToken(tokenString string) map[string]interface{} {
	// Разбираем токен без проверки подписи
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})

	result := make(map[string]interface{})

	if token == nil {
		result["valid"] = false
		result["error"] = "невозможно разобрать токен"
		return result
	}

	// Получаем данные из токена
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		result["valid"] = false // Всегда false, т.к. подпись не проверяется
		result["claims"] = claims

		// Извлекаем полезные поля
		if userID, ok := claims["user_id"].(float64); ok {
			result["user_id"] = uint(userID)
		}
		if email, ok := claims["email"].(string); ok {
			result["email"] = email
		}
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			result["expires_at"] = expTime
			result["expired"] = expTime.Before(time.Now())
		}
		if iat, ok := claims["iat"].(float64); ok {
			issuedAt := time.Unix(int64(iat), 0)
			result["issued_at"] = issuedAt

			// Проверяем инвалидацию
			isInvalidated := false
			var invalidationTime time.Time

			if userID, ok := claims["user_id"].(float64); ok {
				uid := uint(userID)
				s.mu.RLock()
				invTime, exists := s.invalidatedUsers[uid]
				s.mu.RUnlock()

				if exists {
					invalidationTime = invTime
					isInvalidated = !issuedAt.After(invTime)
				}
			}

			result["invalidation_time"] = invalidationTime
			result["is_invalidated"] = isInvalidated
		}
	} else {
		result["error"] = "невозможно извлечь claims из токена"
	}

	return result
}

// GenerateWSTicket создает краткоживущий JWT токен специально для WebSocket подключения
func (s *JWTService) GenerateWSTicket(userID uint, email string) (string, error) {
	claims := &JWTCustomClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * 30)), // 30 секунд жизни
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Добавляем специальный claim для обозначения, что это WS-тикет
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["purpose"] = "websocket_auth"

	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		log.Printf("[JWT] Ошибка генерации WS-тикета для пользователя ID=%d: %v", userID, err)
		return "", err
	}

	log.Printf("[JWT] WS-тикет успешно сгенерирован для пользователя ID=%d, выдан в %v, истекает через 30 секунд",
		userID, claims.IssuedAt)
	return tokenString, nil
}
