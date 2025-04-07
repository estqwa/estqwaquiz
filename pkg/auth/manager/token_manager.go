package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
	"github.com/yourusername/trivia-api/pkg/auth"
)

// Константы для настройки токенов
const (
	// Время жизни access-токена (15 минут)
	AccessTokenLifetime = 15 * time.Minute
	// Время жизни refresh-токена (30 дней)
	RefreshTokenLifetime = 30 * 24 * time.Hour
	// Максимальное количество активных refresh-токенов на пользователя (по умолчанию)
	DefaultMaxRefreshTokensPerUser = 10
	// Имя cookie для refresh-токена
	RefreshTokenCookie = "refresh_token"
	// Имя cookie для access-токена
	AccessTokenCookie = "access_token"
	// Имя заголовка для CSRF токена
	CSRFHeader = "X-CSRF-Token"

	// Время жизни CSRF токена в памяти
	csrfTokenLifetime = 15 * time.Minute

	// Время жизни ключа JWT по умолчанию
	DefaultJWTKeyLifetime = 90 * 24 * time.Hour // 90 дней
)

// TokenErrorType определяет тип ошибки токена
type TokenErrorType string

const (
	// Ошибки генерации токенов
	TokenGenerationFailed TokenErrorType = "TOKEN_GENERATION_FAILED"

	// Ошибки валидации
	InvalidRefreshToken TokenErrorType = "INVALID_REFRESH_TOKEN"
	ExpiredRefreshToken TokenErrorType = "EXPIRED_REFRESH_TOKEN"
	InvalidAccessToken  TokenErrorType = "INVALID_ACCESS_TOKEN"
	ExpiredAccessToken  TokenErrorType = "EXPIRED_ACCESS_TOKEN"
	InvalidCSRFToken    TokenErrorType = "INVALID_CSRF_TOKEN"
	UserNotFound        TokenErrorType = "USER_NOT_FOUND"
	InactiveUser        TokenErrorType = "INACTIVE_USER"

	// Ошибки базы данных или репозитория
	DatabaseError TokenErrorType = "DATABASE_ERROR"

	// Прочие ошибки
	TokenRevoked     TokenErrorType = "TOKEN_REVOKED"
	TooManySessions  TokenErrorType = "TOO_MANY_SESSIONS"
	KeyRotationError TokenErrorType = "KEY_ROTATION_ERROR"
	KeyNotFoundError TokenErrorType = "KEY_NOT_FOUND"
)

// TokenError представляет ошибку при работе с токенами
type TokenError struct {
	Type    TokenErrorType
	Message string
	Err     error
}

// Error возвращает строковое представление ошибки
func (e *TokenError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewTokenError создает новую ошибку токена
func NewTokenError(tokenType TokenErrorType, message string, err error) *TokenError {
	return &TokenError{
		Type:    tokenType,
		Message: message,
		Err:     err,
	}
}

// TokenInfo содержит информацию о сроке действия токенов
type TokenInfo struct {
	AccessTokenExpires   time.Time `json:"access_token_expires"`
	RefreshTokenExpires  time.Time `json:"refresh_token_expires"`
	AccessTokenValidFor  float64   `json:"access_token_valid_for"`
	RefreshTokenValidFor float64   `json:"refresh_token_valid_for"`
}

// CSRFToken содержит данные CSRF токена
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// TokenResponse представляет ответ с токенами
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	CSRFToken   string `json:"csrf_token,omitempty"`
	UserID      uint   `json:"user_id"`
}

// JWTKeyRotation описывает ключ подписи JWT с метаданными
type JWTKeyRotation struct {
	ID        string    // Идентификатор ключа
	Secret    string    // Секретный ключ
	CreatedAt time.Time // Время создания
	ExpiresAt time.Time // Время истечения
	IsActive  bool      // Флаг активности
}

// TokenManager управляет выдачей и валидацией токенов
type TokenManager struct {
	jwtService              *auth.JWTService
	refreshTokenRepo        repository.RefreshTokenRepository
	userRepo                repository.UserRepository
	csrfTokens              map[string]CSRFToken
	csrfMutex               sync.RWMutex
	jwtKeys                 []JWTKeyRotation
	jwtKeysMutex            sync.RWMutex
	currentJWTKeyID         string
	accessTokenExpiry       time.Duration
	refreshTokenExpiry      time.Duration
	maxRefreshTokensPerUser int       // Добавлено: настраиваемый лимит сессий
	lastKeyRotation         time.Time // Добавлено: время последней ротации ключей
	isProductionMode        bool      // Определяет, устанавливать ли Secure флаг для cookies (true в production, false в development)
}

// NewTokenManager создает новый менеджер токенов
func NewTokenManager(
	jwtService *auth.JWTService,
	refreshTokenRepo repository.RefreshTokenRepository,
	userRepo repository.UserRepository,
) *TokenManager {
	if jwtService == nil {
		log.Fatal("JWTService is required for TokenManager")
	}
	if refreshTokenRepo == nil {
		log.Fatal("RefreshTokenRepository is required for TokenManager")
	}
	if userRepo == nil {
		log.Fatal("UserRepository is required for TokenManager")
	}

	// Устанавливаем значения по умолчанию, если они не были заданы
	accessTokenExpiry := 30 * time.Minute     // Можно вынести в конфигурацию
	refreshTokenExpiry := 30 * 24 * time.Hour // Можно вынести в конфигурацию
	maxRefreshTokens := 10                    // Можно вынести в конфигурацию

	tm := &TokenManager{
		jwtService:              jwtService,
		refreshTokenRepo:        refreshTokenRepo,
		userRepo:                userRepo,
		csrfTokens:              make(map[string]CSRFToken),
		jwtKeys:                 make([]JWTKeyRotation, 0),
		accessTokenExpiry:       accessTokenExpiry,
		refreshTokenExpiry:      refreshTokenExpiry,
		maxRefreshTokensPerUser: maxRefreshTokens,
		isProductionMode:        true, // По умолчанию считаем production
	}

	// Запускаем фоновую задачу очистки CSRF токенов
	go tm.cleanupExpiredCSRFTokensLoop()
	// Запускаем фоновую задачу ротации JWT ключей (если необходимо)
	// go tm.jwtKeyRotationLoop() // Раскомментировать, если нужна автоматическая ротация

	// Инициализация JWT ключей при старте
	if err := tm.InitializeJWTKeys(); err != nil {
		log.Printf("Warning: Failed to initialize JWT keys: %v. Using default secret.", err)
		// Продолжаем работу с секретом из jwtService по умолчанию
	}

	return tm
}

// SetAccessTokenExpiry устанавливает время жизни access токена
func (m *TokenManager) SetAccessTokenExpiry(duration time.Duration) {
	if duration > 0 {
		m.accessTokenExpiry = duration
		log.Printf("[TokenManager] Access token expiry set to: %v", duration)
	} else {
		log.Printf("[TokenManager] Warning: Invalid access token expiry duration provided: %v. Using default: %v", duration, m.accessTokenExpiry)
	}
}

// SetRefreshTokenExpiry устанавливает время жизни refresh токена
func (m *TokenManager) SetRefreshTokenExpiry(duration time.Duration) {
	if duration > 0 {
		m.refreshTokenExpiry = duration
		log.Printf("[TokenManager] Refresh token expiry set to: %v", duration)
	} else {
		log.Printf("[TokenManager] Warning: Invalid refresh token expiry duration provided: %v. Using default: %v", duration, m.refreshTokenExpiry)
	}
}

// SetProductionMode устанавливает флаг режима production для Secure cookies
func (m *TokenManager) SetProductionMode(isProduction bool) {
	m.isProductionMode = isProduction
	log.Printf("[TokenManager] Production mode set to: %v", isProduction)
}

// GenerateTokenPair создает новую пару токенов (access и refresh)
// Эта функция теперь использует jwtService напрямую, а не через tokenService
func (m *TokenManager) GenerateTokenPair(userID uint, deviceID, ipAddress, userAgent string) (*TokenResponse, error) {
	user, err := m.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при получении пользователя ID=%d: %v", userID, err)
		return nil, NewTokenError(UserNotFound, "пользователь не найден", err)
	}

	// Генерируем access-токен с использованием текущего активного ключа
	currentKeyID, _, keyErr := m.GetCurrentJWTKey()
	if keyErr != nil {
		log.Printf("[TokenManager] Ошибка получения текущего JWT ключа: %v. Используем дефолтный.", keyErr)
		// Если ключи не настроены, используем секрет из jwtService
	}

	// Генерируем access-токен через jwtService (он сам обработает ключи, если они есть)
	accessToken, err := m.jwtService.GenerateToken(user) // jwtService использует свой секрет по умолчанию или переданные ключи
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации access-токена для пользователя ID=%d: %v", userID, err)
		return nil, NewTokenError(TokenGenerationFailed, "ошибка генерации access токена", err)
	}

	// Генерируем refresh-токен
	_, err = m.generateRefreshToken(userID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации refresh-токена для пользователя ID=%d: %v", userID, err)
		return nil, NewTokenError(TokenGenerationFailed, "ошибка генерации refresh токена", err)
	}

	// Лимитируем количество активных refresh-токенов
	err = m.limitUserSessions(userID)
	if err != nil {
		// Логируем ошибку, но не прерываем процесс выдачи токенов
		log.Printf("[TokenManager] Ошибка при лимитировании сессий пользователя ID=%d: %v", userID, err)
	}

	// Генерируем CSRF токен
	csrfToken := m.generateCSRFToken(userID)

	log.Printf("[TokenManager] Сгенерирована пара токенов для пользователя ID=%d, JWT Key ID: %s", userID, currentKeyID)

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(m.accessTokenExpiry.Seconds()),
		CSRFToken:   csrfToken,
		UserID:      userID,
		// RefreshToken больше не возвращается в ответе, используется кука
	}, nil
}

// RefreshTokens обновляет пару токенов, используя refresh токен
// Эта функция теперь использует jwtService напрямую
func (m *TokenManager) RefreshTokens(refreshToken, csrfToken, deviceID, ipAddress, userAgent string) (*TokenResponse, error) {
	// Валидируем refresh токен
	tokenEntity, err := m.refreshTokenRepo.GetTokenByValue(refreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrExpiredToken) {
			return nil, NewTokenError(InvalidRefreshToken, "недействительный или истекший refresh токен", err)
		}
		log.Printf("[TokenManager] Ошибка при получении refresh-токена: %v", err)
		return nil, NewTokenError(DatabaseError, "ошибка при проверке refresh токена", err)
	}

	// Проверяем срок действия
	if tokenEntity.ExpiresAt.Before(time.Now()) {
		// Помечаем как истекший на всякий случай
		m.refreshTokenRepo.MarkTokenAsExpired(refreshToken) // Игнорируем ошибку здесь
		return nil, NewTokenError(ExpiredRefreshToken, "refresh токен истек", nil)
	}

	// Валидируем CSRF токен
	if !m.validateCSRFToken(tokenEntity.UserID, csrfToken) {
		return nil, NewTokenError(InvalidCSRFToken, "недействительный CSRF токен", nil)
	}

	// Получаем пользователя
	user, err := m.userRepo.GetByID(tokenEntity.UserID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при получении пользователя ID=%d для обновления токенов: %v", tokenEntity.UserID, err)
		return nil, NewTokenError(UserNotFound, "пользователь не найден", err)
	}

	// Помечаем старый refresh токен как истекший
	if err := m.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
		log.Printf("[TokenManager] Ошибка при маркировке старого refresh-токена как истекшего (ID: %d): %v", tokenEntity.ID, err)
		// Не критично, продолжаем
	}

	// Генерируем новый access токен через jwtService
	newAccessToken, err := m.jwtService.GenerateToken(user)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации нового access-токена для пользователя ID=%d: %v", user.ID, err)
		return nil, NewTokenError(TokenGenerationFailed, "ошибка генерации нового access токена", err)
	}

	// Генерируем новый refresh токен
	_, err = m.generateRefreshToken(user.ID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации нового refresh-токена для пользователя ID=%d: %v", user.ID, err)
		return nil, NewTokenError(TokenGenerationFailed, "ошибка генерации нового refresh токена", err)
	}

	// Лимитируем сессии снова
	err = m.limitUserSessions(user.ID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при лимитировании сессий пользователя ID=%d после обновления: %v", user.ID, err)
	}

	// Генерируем новый CSRF токен
	newCSRFToken := m.generateCSRFToken(user.ID)

	log.Printf("[TokenManager] Обновлена пара токенов для пользователя ID=%d", user.ID)

	return &TokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(m.accessTokenExpiry.Seconds()),
		CSRFToken:   newCSRFToken,
		UserID:      user.ID,
		// RefreshToken больше не возвращается в ответе, используется кука
	}, nil
}

// GetTokenInfo возвращает информацию о сроках действия текущих токенов
func (m *TokenManager) GetTokenInfo(refreshToken string) (*TokenInfo, error) {
	// Находим refresh-токен в БД
	token, err := m.refreshTokenRepo.GetTokenByValue(refreshToken)
	if err != nil {
		return nil, NewTokenError(InvalidRefreshToken, "Недействительный refresh-токен", err)
	}

	// Вычисляем время истечения access-токена (примерно)
	accessTokenExpires := time.Now().Add(m.accessTokenExpiry)

	now := time.Now()
	return &TokenInfo{
		AccessTokenExpires:   accessTokenExpires,
		RefreshTokenExpires:  token.ExpiresAt,
		AccessTokenValidFor:  accessTokenExpires.Sub(now).Seconds(),
		RefreshTokenValidFor: token.ExpiresAt.Sub(now).Seconds(),
	}, nil
}

// RevokeRefreshToken отзывает (помечает как истекший) указанный refresh токен
func (m *TokenManager) RevokeRefreshToken(refreshToken string) error {
	if err := m.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
		// Проверяем, была ли ошибка "не найдено"
		if errors.Is(err, repository.ErrNotFound) {
			log.Printf("[TokenManager] Попытка отозвать несуществующий refresh токен.")
			return NewTokenError(InvalidRefreshToken, "токен не найден", err) // Возвращаем ошибку недействительного токена
		}
		log.Printf("[TokenManager] Ошибка при отзыве refresh-токена: %v", err)
		return NewTokenError(DatabaseError, "ошибка при отзыве токена", err)
	}

	log.Printf("[TokenManager] Отозван refresh-токен")
	return nil
}

// RevokeAllUserTokens отзывает все refresh-токены пользователя
func (m *TokenManager) RevokeAllUserTokens(userID uint) error {
	// Помечаем все refresh-токены пользователя как истекшие
	if err := m.refreshTokenRepo.MarkAllAsExpiredForUser(userID); err != nil {
		log.Printf("[TokenManager] Ошибка при отзыве всех refresh-токенов пользователя ID=%d: %v", userID, err)
		// Даже если произошла ошибка с refresh токенами, пытаемся инвалидировать JWT
		if jwtErr := m.jwtService.InvalidateTokensForUser(context.Background(), userID); jwtErr != nil {
			log.Printf("[TokenManager] Дополнительная ошибка при инвалидации JWT токенов пользователя ID=%d: %v", userID, jwtErr)
		}
		return NewTokenError(DatabaseError, "ошибка отзыва refresh токенов", err)
	}

	// Дополнительно инвалидируем JWT после успешного отзыва refresh токенов
	if jwtErr := m.jwtService.InvalidateTokensForUser(context.Background(), userID); jwtErr != nil {
		log.Printf("[TokenManager] Ошибка при инвалидации JWT токенов пользователя ID=%d после отзыва refresh токенов: %v", userID, jwtErr)
		// Не возвращаем ошибку JWT как критическую, так как refresh уже отозваны
	}

	// Удаляем все CSRF токены пользователя
	m.csrfMutex.Lock()
	defer m.csrfMutex.Unlock()

	for k, _ := range m.csrfTokens {
		parts := strings.Split(k, ":")
		if len(parts) == 2 && parts[0] == fmt.Sprintf("%d", userID) {
			delete(m.csrfTokens, k)
		}
	}

	log.Printf("[TokenManager] Отозваны все токены пользователя ID=%d", userID)
	return nil
}

// GetUserActiveSessions возвращает список активных сессий (refresh токенов) для пользователя
func (m *TokenManager) GetUserActiveSessions(userID uint) ([]entity.RefreshToken, error) {
	tokensPtr, err := m.refreshTokenRepo.GetActiveTokensForUser(userID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при получении активных сессий пользователя ID=%d: %v", userID, err)
		return nil, NewTokenError(DatabaseError, "ошибка получения сессий", err)
	}

	// Преобразуем []*entity.RefreshToken в []entity.RefreshToken
	tokens := make([]entity.RefreshToken, len(tokensPtr))
	for i, t := range tokensPtr {
		tokens[i] = *t
	}

	log.Printf("[TokenManager] Получено %d активных токенов пользователя ID=%d", len(tokens), userID)
	return tokens, nil
}

// SetRefreshTokenCookie устанавливает refresh-токен в HttpOnly куки
func (m *TokenManager) SetRefreshTokenCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.isProductionMode,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(m.refreshTokenExpiry.Seconds()),
	})
}

// SetAccessTokenCookie устанавливает access-токен в HttpOnly куки
func (m *TokenManager) SetAccessTokenCookie(w http.ResponseWriter, accessToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.isProductionMode,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(m.accessTokenExpiry.Seconds()),
	})
}

// GetRefreshTokenFromCookie получает refresh-токен из куки
func (m *TokenManager) GetRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", NewTokenError(InvalidRefreshToken, "кука refresh_token не найдена", err)
		}
		return "", NewTokenError(InvalidRefreshToken, "ошибка чтения куки refresh_token", err)
	}
	return cookie.Value, nil
}

// GetAccessTokenFromCookie получает access-токен из куки
func (m *TokenManager) GetAccessTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(AccessTokenCookie)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", NewTokenError(InvalidAccessToken, "кука access_token не найдена", err)
		}
		return "", NewTokenError(InvalidAccessToken, "ошибка чтения куки access_token", err)
	}
	return cookie.Value, nil
}

// ClearRefreshTokenCookie удаляет cookie с refresh-токеном
func (m *TokenManager) ClearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.isProductionMode,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// ClearAccessTokenCookie удаляет cookie с access-токеном
func (m *TokenManager) ClearAccessTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.isProductionMode,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// VerifyCSRFToken проверяет CSRF токен
func (m *TokenManager) VerifyCSRFToken(userID uint, csrfToken string) bool {
	return m.validateCSRFToken(userID, csrfToken)
}

// CleanupExpiredTokens удаляет все истекшие refresh-токены и CSRF токены
func (m *TokenManager) CleanupExpiredTokens() error {
	// Очищаем истекшие refresh-токены в БД
	count, err := m.refreshTokenRepo.CleanupExpiredTokens()
	if err != nil {
		log.Printf("[TokenManager] Ошибка при очистке истекших refresh-токенов: %v", err)
		// Очистка JWT инвалидаций
		if jwtErr := m.jwtService.CleanupInvalidatedUsers(context.Background()); jwtErr != nil {
			log.Printf("[TokenManager] Дополнительная ошибка при очистке инвалидированных JWT токенов: %v", jwtErr)
		}
		return NewTokenError(DatabaseError, "ошибка очистки истекших токенов", err)
	}

	// Очищаем истекшие CSRF токены
	m.cleanupExpiredCSRFTokens()

	// Для обратной совместимости также запускаем очистку инвалидированных JWT-токенов
	if err := m.jwtService.CleanupInvalidatedUsers(context.Background()); err != nil {
		log.Printf("[TokenManager] Ошибка при очистке инвалидированных JWT токенов: %v", err)
		// Не возвращаем ошибку, так как основная очистка прошла
	}

	log.Printf("[TokenManager] Выполнена очистка %d истекших токенов", count)
	return nil
}

// RotateJWTKeys выполняет ротацию ключей подписи JWT
func (m *TokenManager) RotateJWTKeys() (string, error) {
	// Убрали проверку пользователя, т.к. ротация - системная операция
	// _, err := m.userRepo.GetByID(userID)
	// if err != nil {
	// 	log.Printf("[TokenManager] Пользователь ID=%d не найден при генерации ключа JWT", userID)
	// 	return "", NewTokenError(UserNotFound, "пользователь не найден", err)
	// }

	// Генерируем новый секрет
	newSecret := generateRandomString(64)
	newKeyID := generateRandomString(16)
	now := time.Now()
	// Используем константу для времени жизни ключа
	expiry := now.Add(DefaultJWTKeyLifetime)

	newKey := JWTKeyRotation{
		ID:        newKeyID,
		Secret:    newSecret,
		CreatedAt: now,
		ExpiresAt: expiry,
		IsActive:  true,
	}

	m.jwtKeysMutex.Lock()
	// Деактивируем текущий активный ключ (если есть)
	for i := range m.jwtKeys {
		if m.jwtKeys[i].IsActive {
			m.jwtKeys[i].IsActive = false
			break
		}
	}
	// Добавляем новый ключ
	m.jwtKeys = append(m.jwtKeys, newKey)
	m.currentJWTKeyID = newKeyID
	m.lastKeyRotation = now
	m.jwtKeysMutex.Unlock()

	// TODO: Добавить логику сохранения ключей в персистентное хранилище (БД, файл)
	// сейчас ключи хранятся только в памяти и теряются при перезапуске
	log.Printf("[TokenManager] Успешно сгенерирован и активирован новый JWT ключ ID: %s", newKeyID)

	return newKeyID, nil
}

// Служебные функции

// generateRefreshToken генерирует новый refresh-токен и сохраняет его в БД
func (m *TokenManager) generateRefreshToken(userID uint, deviceID, ipAddress, userAgent string) (string, error) {
	// Генерируем случайный токен
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	tokenString := hex.EncodeToString(randomBytes)

	// Время истечения - "скользящее окно" 30 дней от текущего момента
	expiresAt := time.Now().Add(m.refreshTokenExpiry)

	// Создаем запись в БД
	token := entity.NewRefreshToken(userID, tokenString, deviceID, ipAddress, userAgent, expiresAt)

	// Сохраняем в БД
	_, err := m.refreshTokenRepo.CreateToken(token)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// generateCSRFToken генерирует CSRF токен для пользователя
func (m *TokenManager) generateCSRFToken(userID uint) string {
	// Генерируем случайный токен
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("[TokenManager] Ошибка при генерации CSRF токена: %v", err)
		return ""
	}
	csrfToken := hex.EncodeToString(randomBytes)

	// Сохраняем токен в памяти с привязкой к пользователю
	m.csrfMutex.Lock()
	defer m.csrfMutex.Unlock()

	key := fmt.Sprintf("%d:%s", userID, csrfToken)
	m.csrfTokens[key] = CSRFToken{
		Token:     csrfToken,
		ExpiresAt: time.Now().Add(m.accessTokenExpiry),
	}

	return csrfToken
}

// validateCSRFToken проверяет CSRF токен
func (m *TokenManager) validateCSRFToken(userID uint, csrfToken string) bool {
	if csrfToken == "" {
		return false
	}

	m.csrfMutex.RLock()
	defer m.csrfMutex.RUnlock()

	key := fmt.Sprintf("%d:%s", userID, csrfToken)
	token, exists := m.csrfTokens[key]
	if !exists {
		return false
	}

	// Проверяем, не истек ли токен
	if token.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// cleanupExpiredCSRFTokens удаляет истекшие CSRF токены
func (m *TokenManager) cleanupExpiredCSRFTokens() {
	m.csrfMutex.Lock()
	defer m.csrfMutex.Unlock()

	now := time.Now()
	for k, v := range m.csrfTokens {
		if v.ExpiresAt.Before(now) {
			delete(m.csrfTokens, k)
		}
	}
}

// generateNewJWTKey генерирует новый ключ подписи JWT
func (m *TokenManager) generateNewJWTKey() (string, error) {
	// Генерируем случайный ключ
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(randomBytes)
	keyID := fmt.Sprintf("key-%d", time.Now().UnixNano())

	// Добавляем новый ключ
	m.jwtKeysMutex.Lock()
	defer m.jwtKeysMutex.Unlock()

	// Создаем новый ключ
	newKey := JWTKeyRotation{
		ID:        keyID,
		Secret:    secret,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // 90 дней
		IsActive:  true,
	}

	// Добавляем в список ключей
	m.jwtKeys = append(m.jwtKeys, newKey)
	m.currentJWTKeyID = keyID

	return keyID, nil
}

// deactivateOldJWTKeys помечает старые (неактивные и истекшие) ключи
func (m *TokenManager) deactivateOldJWTKeys() {
	m.jwtKeysMutex.Lock()
	defer m.jwtKeysMutex.Unlock()

	// Деактивируем ключи старше 60 дней
	cutoffTime := time.Now().Add(-60 * 24 * time.Hour)
	for i := range m.jwtKeys {
		if m.jwtKeys[i].CreatedAt.Before(cutoffTime) {
			m.jwtKeys[i].IsActive = false
		}
	}
}

// GetCurrentJWTKey возвращает текущий активный ключ подписи JWT
func (m *TokenManager) GetCurrentJWTKey() (string, string, error) {
	m.jwtKeysMutex.RLock()
	defer m.jwtKeysMutex.RUnlock()

	// Если нет ключей, генерируем новый
	if len(m.jwtKeys) == 0 || m.currentJWTKeyID == "" {
		m.jwtKeysMutex.RUnlock()
		_, err := m.generateNewJWTKey()
		if err != nil {
			return "", "", err
		}
		m.jwtKeysMutex.RLock()
	}

	// Ищем текущий ключ
	for _, key := range m.jwtKeys {
		if key.ID == m.currentJWTKeyID && key.IsActive {
			return key.ID, key.Secret, nil
		}
	}

	// Если текущий ключ не найден или не активен, ищем любой активный
	for _, key := range m.jwtKeys {
		if key.IsActive {
			m.currentJWTKeyID = key.ID
			return key.ID, key.Secret, nil
		}
	}

	// Если нет активных ключей, генерируем новый (после разблокировки мьютекса)
	m.jwtKeysMutex.RUnlock()
	_, err := m.generateNewJWTKey()
	if err != nil {
		return "", "", err
	}
	m.jwtKeysMutex.RLock()

	// Находим новый ключ
	for _, key := range m.jwtKeys {
		if key.ID == m.currentJWTKeyID {
			return key.ID, key.Secret, nil
		}
	}

	return "", "", errors.New("не удалось найти или создать активный ключ JWT")
}

// SetMaxRefreshTokensPerUser устанавливает максимальное количество активных сессий для пользователя
func (m *TokenManager) SetMaxRefreshTokensPerUser(limit int) {
	if limit <= 0 {
		limit = DefaultMaxRefreshTokensPerUser
	}
	m.maxRefreshTokensPerUser = limit
	log.Printf("[TokenManager] Установлен лимит активных сессий: %d", limit)
}

// GetMaxRefreshTokensPerUser возвращает текущий лимит активных сессий
func (m *TokenManager) GetMaxRefreshTokensPerUser() int {
	return m.maxRefreshTokensPerUser
}

// InitializeJWTKeys инициализирует ключи подписи JWT при запуске
func (m *TokenManager) InitializeJWTKeys() error {
	// Добавляем ключ по умолчанию из конфигурации jwtService
	// TODO: Получать секрет из конфигурации, а не напрямую из jwtService?
	defaultSecret := "" // Нужно получить секрет из jwtService или конфига
	if m.jwtService != nil {
		// defaultSecret = m.jwtService.GetSecret() // Примерный вызов
		// Пытаемся получить секрет из самого сервиса JWT, если он его хранит (нужен метод GetSecret)
		// В текущей реализации jwtService секрет приватный. Мы можем его либо передать
		// при инициализации TokenManager, либо загрузить из конфигурации.
		// Пока оставим пустым и будем полагаться на ключ из GenerateToken, если Initialize не сработает.
		log.Println("[TokenManager] Не удалось получить секрет по умолчанию из jwtService для инициализации ключей. Используйте конфигурацию или передайте секрет явно.")
	}

	if defaultSecret == "" {
		log.Println("[TokenManager] Предупреждение: Не удалось инициализировать JWT ключи из-за отсутствия секрета по умолчанию.")
		return nil // Не критично, если jwtService сам использует свой секрет
	}

	initialKey := JWTKeyRotation{
		ID:        "initial-key",
		Secret:    defaultSecret,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // 90 дней
		IsActive:  true,
	}

	m.jwtKeysMutex.Lock()
	m.jwtKeys = append(m.jwtKeys, initialKey)
	m.currentJWTKeyID = "initial-key"
	m.jwtKeysMutex.Unlock()

	log.Printf("[TokenManager] Успешно сгенерирован и активирован новый JWT ключ ID: %s", "initial-key")

	return nil
}

// CheckKeyRotation проверяет, нужно ли выполнить ротацию ключей JWT
func (m *TokenManager) CheckKeyRotation() bool {
	// Выполняем ротацию ключей раз в месяц
	if time.Since(m.lastKeyRotation) > 30*24*time.Hour {
		log.Printf("[TokenManager] Проверка ротации ключей: пора выполнить ротацию (последняя была %s)", m.lastKeyRotation)
		_, err := m.RotateJWTKeys()
		if err != nil {
			log.Printf("[TokenManager] Ошибка при автоматической ротации ключей: %v", err)
			return false
		}
		m.lastKeyRotation = time.Now()
		return true
	}
	return false
}

// GetActiveJWTKeys возвращает все активные ключи JWT
func (m *TokenManager) GetActiveJWTKeys() []JWTKeyRotation {
	m.jwtKeysMutex.RLock()
	defer m.jwtKeysMutex.RUnlock()

	activeKeys := make([]JWTKeyRotation, 0)
	for _, key := range m.jwtKeys {
		if key.IsActive {
			// Создаем копию без секретного ключа для безопасности
			keyCopy := JWTKeyRotation{
				ID:        key.ID,
				CreatedAt: key.CreatedAt,
				ExpiresAt: key.ExpiresAt,
				IsActive:  key.IsActive,
			}
			activeKeys = append(activeKeys, keyCopy)
		}
	}

	return activeKeys
}

// GetJWTKeySummary возвращает сводку по ключам JWT
func (m *TokenManager) GetJWTKeySummary() map[string]interface{} {
	m.jwtKeysMutex.RLock()
	defer m.jwtKeysMutex.RUnlock()

	activeCount := 0
	inactiveCount := 0
	var oldestKey time.Time
	var newestKey time.Time

	if len(m.jwtKeys) > 0 {
		oldestKey = m.jwtKeys[0].CreatedAt
		newestKey = m.jwtKeys[0].CreatedAt
	}

	for _, key := range m.jwtKeys {
		if key.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}

		if key.CreatedAt.Before(oldestKey) {
			oldestKey = key.CreatedAt
		}
		if key.CreatedAt.After(newestKey) {
			newestKey = key.CreatedAt
		}
	}

	return map[string]interface{}{
		"active_keys":     activeCount,
		"inactive_keys":   inactiveCount,
		"total_keys":      len(m.jwtKeys),
		"current_key_id":  m.currentJWTKeyID,
		"last_rotation":   m.lastKeyRotation,
		"oldest_key_date": oldestKey,
		"newest_key_date": newestKey,
	}
}

// Добавляем хелпер для лимитирования сессий, чтобы избежать дублирования кода
func (m *TokenManager) limitUserSessions(userID uint) error {
	count, err := m.refreshTokenRepo.CountTokensForUser(userID)
	if err != nil {
		return fmt.Errorf("ошибка подсчета токенов: %w", err)
	}

	if count > m.maxRefreshTokensPerUser {
		log.Printf("[TokenManager] Превышен лимит сессий для пользователя ID=%d (%d > %d). Удаление старых.", userID, count, m.maxRefreshTokensPerUser)
		if err := m.refreshTokenRepo.MarkOldestAsExpiredForUser(userID, m.maxRefreshTokensPerUser); err != nil {
			return fmt.Errorf("ошибка маркировки старых токенов: %w", err)
		}
	}
	return nil
}

// cleanupExpiredCSRFTokensLoop запускает периодическую очистку CSRF токенов
func (m *TokenManager) cleanupExpiredCSRFTokensLoop() {
	ticker := time.NewTicker(1 * time.Hour) // Запускаем очистку каждый час
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredCSRFTokens()
	}
}

// jwtKeyRotationLoop запускает периодическую ротацию JWT ключей
// func (m *TokenManager) jwtKeyRotationLoop() { // Раскомментировать если нужна авторотация
// 	// Определяем интервал ротации (например, каждые 24 часа)
// 	rotationInterval := 24 * time.Hour
// 	ticker := time.NewTicker(rotationInterval)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		if _, err := m.RotateJWTKeys(); err != nil {
// 			log.Printf("[TokenManager] Ошибка автоматической ротации JWT ключей: %v", err)
// 		} else {
// 			log.Println("[TokenManager] Автоматическая ротация JWT ключей выполнена успешно")
// 			m.deactivateOldJWTKeys() // Деактивируем старые ключи после успешной ротации
// 		}
// 	}
// } // Раскомментировать если нужна авторотация

// generateRandomString генерирует случайную строку указанной длины в hex формате
func generateRandomString(length int) string {
	b := make([]byte, length/2) // Каждый байт кодируется двумя hex символами
	if _, err := rand.Read(b); err != nil {
		// В реальном приложении здесь должна быть более надежная обработка ошибки,
		// возможно, паника, так как генерация секретов критична.
		log.Printf("CRITICAL: Ошибка генерации случайных байт: %v", err)
		panic(fmt.Sprintf("Failed to generate random string: %v", err))
	}
	return hex.EncodeToString(b)
}
