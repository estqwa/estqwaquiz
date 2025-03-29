package manager

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
)

// TokenErrorType определяет тип ошибки токена
type TokenErrorType string

const (
	TokenErrorExpired      TokenErrorType = "token_expired"
	TokenErrorInvalid      TokenErrorType = "token_invalid"
	TokenErrorMalformed    TokenErrorType = "token_malformed"
	TokenErrorUnauthorized TokenErrorType = "unauthorized"
	TokenErrorCSRFMismatch TokenErrorType = "csrf_mismatch"
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
	tokenService            *auth.TokenService
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
	tokenService *auth.TokenService,
	refreshTokenRepo repository.RefreshTokenRepository,
	userRepo repository.UserRepository,
) *TokenManager {
	// Определяем режим работы по Gin. В production режиме gin.Mode() == gin.ReleaseMode
	isProductionMode := gin.Mode() == gin.ReleaseMode
	log.Printf("[TokenManager] Инициализация в режиме: %s (production mode: %v)", gin.Mode(), isProductionMode)

	return &TokenManager{
		jwtService:              jwtService,
		tokenService:            tokenService,
		refreshTokenRepo:        refreshTokenRepo,
		userRepo:                userRepo,
		csrfTokens:              make(map[string]CSRFToken),
		jwtKeys:                 make([]JWTKeyRotation, 0),
		accessTokenExpiry:       AccessTokenLifetime,
		refreshTokenExpiry:      RefreshTokenLifetime,
		maxRefreshTokensPerUser: DefaultMaxRefreshTokensPerUser,
		lastKeyRotation:         time.Now(),
		isProductionMode:        isProductionMode,
	}
}

// GenerateTokenPair создает новую пару токенов для пользователя
func (m *TokenManager) GenerateTokenPair(userID uint, deviceID, ipAddress, userAgent string) (*TokenResponse, error) {
	// Получаем пользователя из БД
	user, err := m.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при получении пользователя ID=%d: %v", userID, err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Пользователь не найден", err)
	}

	// Генерируем access-токен
	accessToken, err := m.jwtService.GenerateToken(user)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации access-токена: %v", err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Ошибка генерации токена", err)
	}

	// Генерируем refresh-токен
	_, err = m.generateRefreshToken(userID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации refresh-токена: %v", err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Ошибка генерации refresh-токена", err)
	}

	// Лимитируем количество refresh-токенов для пользователя
	count, err := m.refreshTokenRepo.CountTokensForUser(userID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при подсчете refresh-токенов: %v", err)
		// Не возвращаем ошибку, продолжаем
	} else if count > m.maxRefreshTokensPerUser {
		// Помечаем лишние токены как истекшие, оставляя только maxRefreshTokensPerUser
		if err := m.refreshTokenRepo.MarkOldestAsExpiredForUser(userID, m.maxRefreshTokensPerUser); err != nil {
			log.Printf("[TokenManager] Ошибка при лимитировании refresh-токенов: %v", err)
			// Не возвращаем ошибку, продолжаем
		}
	}

	// Генерируем CSRF токен для защиты от CSRF атак
	csrfToken := m.generateCSRFToken(userID)

	log.Printf("[TokenManager] Сгенерирована пара токенов для пользователя ID=%d", userID)
	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(m.accessTokenExpiry.Seconds()),
		CSRFToken:   csrfToken,
		UserID:      userID,
	}, nil
}

// RefreshTokens обновляет пару токенов по refresh-токену
func (m *TokenManager) RefreshTokens(refreshToken, csrfToken, deviceID, ipAddress, userAgent string) (*TokenResponse, error) {
	// Находим refresh-токен в БД
	token, err := m.refreshTokenRepo.GetTokenByValue(refreshToken)
	if err != nil {
		if err == repository.ErrExpiredToken {
			log.Printf("[TokenManager] Refresh-токен помечен как истекший")
			return nil, NewTokenError(TokenErrorExpired, "Недействительный refresh-токен", err)
		}
		if err == repository.ErrNotFound {
			log.Printf("[TokenManager] Refresh-токен не найден")
			return nil, NewTokenError(TokenErrorInvalid, "Недействительный refresh-токен", err)
		}
		log.Printf("[TokenManager] Ошибка при получении refresh-токена: %v", err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Ошибка при проверке токена", err)
	}

	// Проверяем CSRF токен
	if !m.validateCSRFToken(token.UserID, csrfToken) {
		log.Printf("[TokenManager] Неверный CSRF токен для пользователя ID=%d", token.UserID)
		return nil, NewTokenError(TokenErrorCSRFMismatch, "Неверный CSRF токен", nil)
	}

	// Проверяем, не истек ли токен
	if token.ExpiresAt.Before(time.Now()) {
		log.Printf("[TokenManager] Refresh-токен истек для пользователя ID=%d", token.UserID)
		// Помечаем истекший токен
		if err := m.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
			log.Printf("[TokenManager] Ошибка при маркировке истекшего refresh-токена: %v", err)
		}
		return nil, NewTokenError(TokenErrorExpired, "Refresh-токен истек", nil)
	}

	// Проверяем метаданные устройства для предотвращения кражи токена
	if deviceID != "" && deviceID != token.DeviceID {
		log.Printf("[TokenManager] Несоответствие deviceID: ожидалось %s, получено %s", token.DeviceID, deviceID)
		return nil, NewTokenError(TokenErrorUnauthorized, "Несоответствие идентификатора устройства", nil)
	}

	// Получаем пользователя
	user, err := m.userRepo.GetByID(token.UserID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при получении пользователя ID=%d: %v", token.UserID, err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Пользователь не найден", err)
	}

	// Помечаем использованный refresh-токен как истекший
	if err := m.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
		log.Printf("[TokenManager] Ошибка при маркировке использованного refresh-токена: %v", err)
		// Не возвращаем ошибку, продолжаем
	}

	// Генерируем новые токены
	newAccessToken, err := m.jwtService.GenerateToken(user)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации нового access-токена: %v", err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Ошибка генерации токена", err)
	}

	// "Скользящее окно" - продлеваем срок действия refresh-токена
	_, err = m.generateRefreshToken(user.ID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Printf("[TokenManager] Ошибка генерации нового refresh-токена: %v", err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Ошибка генерации refresh-токена", err)
	}

	// Генерируем новый CSRF токен
	newCsrfToken := m.generateCSRFToken(user.ID)

	log.Printf("[TokenManager] Обновлена пара токенов для пользователя ID=%d", user.ID)
	return &TokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(m.accessTokenExpiry.Seconds()),
		CSRFToken:   newCsrfToken,
		UserID:      user.ID,
	}, nil
}

// GetTokenInfo возвращает информацию о сроках действия текущих токенов
func (m *TokenManager) GetTokenInfo(refreshToken string) (*TokenInfo, error) {
	// Находим refresh-токен в БД
	token, err := m.refreshTokenRepo.GetTokenByValue(refreshToken)
	if err != nil {
		return nil, NewTokenError(TokenErrorInvalid, "Недействительный refresh-токен", err)
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

// RevokeRefreshToken отзывает определенный refresh-токен
func (m *TokenManager) RevokeRefreshToken(refreshToken string) error {
	// Помечаем токен как истекший вместо удаления
	if err := m.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
		log.Printf("[TokenManager] Ошибка при отзыве refresh-токена: %v", err)
		return NewTokenError(TokenErrorUnauthorized, "Ошибка отзыва токена", err)
	}

	log.Printf("[TokenManager] Отозван refresh-токен")
	return nil
}

// RevokeAllUserTokens отзывает все refresh-токены пользователя
func (m *TokenManager) RevokeAllUserTokens(userID uint) error {
	// Помечаем все refresh-токены пользователя как истекшие
	if err := m.refreshTokenRepo.MarkAllAsExpiredForUser(userID); err != nil {
		log.Printf("[TokenManager] Ошибка при отзыве всех refresh-токенов пользователя ID=%d: %v", userID, err)
		return NewTokenError(TokenErrorUnauthorized, "Ошибка отзыва токенов", err)
	}

	// Для обратной совместимости также инвалидируем JWT-токены
	if err := m.jwtService.InvalidateTokensForUser(userID); err != nil {
		log.Printf("[TokenManager] Ошибка при инвалидации JWT-токенов пользователя ID=%d: %v", userID, err)
		// Не возвращаем ошибку, продолжаем (у нас уже отозваны refresh-токены)
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

// GetUserActiveSessions возвращает все активные refresh-токены пользователя
func (m *TokenManager) GetUserActiveSessions(userID uint) ([]entity.RefreshToken, error) {
	tokensPtr, err := m.refreshTokenRepo.GetActiveTokensForUser(userID)
	if err != nil {
		log.Printf("[TokenManager] Ошибка при получении активных токенов пользователя ID=%d: %v", userID, err)
		return nil, NewTokenError(TokenErrorUnauthorized, "Ошибка при получении списка сессий", err)
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
		if err == http.ErrNoCookie {
			return "", NewTokenError(TokenErrorUnauthorized, "Refresh токен отсутствует", err)
		}
		return "", NewTokenError(TokenErrorUnauthorized, "Ошибка при получении refresh токена", err)
	}
	return cookie.Value, nil
}

// GetAccessTokenFromCookie получает access-токен из куки
func (m *TokenManager) GetAccessTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(AccessTokenCookie)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", NewTokenError(TokenErrorUnauthorized, "Access токен отсутствует", err)
		}
		return "", NewTokenError(TokenErrorUnauthorized, "Ошибка при получении access токена", err)
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
		return err
	}

	// Очищаем истекшие CSRF токены
	m.cleanupExpiredCSRFTokens()

	// Для обратной совместимости также запускаем очистку инвалидированных JWT-токенов
	m.jwtService.CleanupInvalidatedUsers()

	log.Printf("[TokenManager] Выполнена очистка %d истекших токенов", count)
	return nil
}

// RotateJWTKeys выполняет ротацию ключей подписи JWT
func (m *TokenManager) RotateJWTKeys() (string, error) {
	// Генерируем новый ключ
	keyID, err := m.generateNewJWTKey()
	if err != nil {
		log.Printf("[TokenManager] Ошибка при генерации нового ключа JWT: %v", err)
		return "", err
	}

	// Деактивируем старые ключи, если они слишком старые
	m.deactivateOldJWTKeys()

	// Обновляем время последней ротации
	m.lastKeyRotation = time.Now()

	log.Printf("[TokenManager] Выполнена ротация ключей JWT, новый ключ: %s", keyID)
	return keyID, nil
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

// deactivateOldJWTKeys деактивирует старые ключи подписи JWT
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
	// Генерируем первый ключ, если нет ключей
	if len(m.jwtKeys) == 0 {
		log.Printf("[TokenManager] Инициализация JWT ключей: генерация первого ключа")
		_, err := m.generateNewJWTKey()
		if err != nil {
			return err
		}
		m.lastKeyRotation = time.Now()
	}
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
