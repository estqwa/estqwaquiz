package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
)

const (
	// Максимальное количество активных refresh-токенов на пользователя
	MaxRefreshTokensPerUser = 10
	// Время жизни refresh-токена (30 дней)
	RefreshTokenLifetime = 30 * 24 * time.Hour
	// Время жизни access-токена (30 минут)
	AccessTokenLifetime = 30 * time.Minute
)

// TokenService предоставляет методы для работы с токенами авторизации
type TokenService struct {
	jwtService         *JWTService
	refreshTokenRepo   repository.RefreshTokenRepository
	userRepo           repository.UserRepository
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewTokenService создает новый сервис токенов
func NewTokenService(
	jwtService *JWTService,
	refreshTokenRepo repository.RefreshTokenRepository,
	userRepo repository.UserRepository,
) *TokenService {
	return &TokenService{
		jwtService:         jwtService,
		refreshTokenRepo:   refreshTokenRepo,
		userRepo:           userRepo,
		accessTokenExpiry:  AccessTokenLifetime,
		refreshTokenExpiry: RefreshTokenLifetime,
	}
}

// TokenPair содержит пару токенов аутентификации
type TokenPair struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
}

// TokenInfo содержит информацию о сроке действия токенов
type TokenInfo struct {
	AccessTokenExpires  time.Time `json:"access_token_expires"`
	RefreshTokenExpires time.Time `json:"refresh_token_expires"`
}

// GenerateTokenPair создает новую пару токенов для пользователя
func (s *TokenService) GenerateTokenPair(userID uint, deviceID, ipAddress, userAgent string) (*TokenPair, error) {
	// Получаем пользователя из БД
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("[TokenService] Ошибка при получении пользователя ID=%d: %v", userID, err)
		return nil, errors.New("пользователь не найден")
	}

	// Генерируем access-токен
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		log.Printf("[TokenService] Ошибка генерации access-токена: %v", err)
		return nil, errors.New("ошибка генерации токена")
	}

	// Генерируем refresh-токен
	refreshToken, err := s.generateRefreshToken(userID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Printf("[TokenService] Ошибка генерации refresh-токена: %v", err)
		return nil, errors.New("ошибка генерации refresh-токена")
	}

	// Лимитируем количество refresh-токенов для пользователя
	count, err := s.refreshTokenRepo.CountTokensForUser(userID)
	if err != nil {
		log.Printf("[TokenService] Ошибка при подсчете refresh-токенов: %v", err)
		// Не возвращаем ошибку, продолжаем
	} else if count > MaxRefreshTokensPerUser {
		// Помечаем лишние токены как истекшие, оставляя только MaxRefreshTokensPerUser
		if err := s.refreshTokenRepo.MarkOldestAsExpiredForUser(userID, MaxRefreshTokensPerUser); err != nil {
			log.Printf("[TokenService] Ошибка при лимитировании refresh-токенов: %v", err)
			// Не возвращаем ошибку, продолжаем
		}
	}

	log.Printf("[TokenService] Сгенерирована пара токенов для пользователя ID=%d", userID)
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.accessTokenExpiry / time.Second,
	}, nil
}

// RefreshTokens обновляет пару токенов по refresh-токену
func (s *TokenService) RefreshTokens(refreshToken, deviceID, ipAddress, userAgent string) (*TokenPair, error) {
	// Находим refresh-токен в БД
	token, err := s.refreshTokenRepo.GetTokenByValue(refreshToken)
	if err != nil {
		if err == repository.ErrExpiredToken {
			log.Printf("[TokenService] Refresh-токен помечен как истекший")
			return nil, errors.New("недействительный refresh-токен")
		}
		if err == repository.ErrNotFound {
			log.Printf("[TokenService] Refresh-токен не найден")
			return nil, errors.New("недействительный refresh-токен")
		}
		log.Printf("[TokenService] Ошибка при получении refresh-токена: %v", err)
		return nil, errors.New("ошибка при проверке токена")
	}

	// Проверяем, не истек ли токен
	if token.ExpiresAt.Before(time.Now()) {
		log.Printf("[TokenService] Refresh-токен истек для пользователя ID=%d", token.UserID)
		// Помечаем истекший токен
		if err := s.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
			log.Printf("[TokenService] Ошибка при маркировке истекшего refresh-токена: %v", err)
		}
		return nil, errors.New("refresh-токен истек")
	}

	// Получаем пользователя
	user, err := s.userRepo.GetByID(token.UserID)
	if err != nil {
		log.Printf("[TokenService] Ошибка при получении пользователя ID=%d: %v", token.UserID, err)
		return nil, errors.New("пользователь не найден")
	}

	// Помечаем использованный refresh-токен как истекший
	if err := s.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
		log.Printf("[TokenService] Ошибка при маркировке использованного refresh-токена: %v", err)
		// Не возвращаем ошибку, продолжаем
	}

	// Генерируем новые токены
	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		log.Printf("[TokenService] Ошибка генерации нового access-токена: %v", err)
		return nil, errors.New("ошибка генерации токена")
	}

	newRefreshToken, err := s.generateRefreshToken(user.ID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Printf("[TokenService] Ошибка генерации нового refresh-токена: %v", err)
		return nil, errors.New("ошибка генерации refresh-токена")
	}

	log.Printf("[TokenService] Обновлена пара токенов для пользователя ID=%d", user.ID)
	return &TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.accessTokenExpiry / time.Second,
	}, nil
}

// CheckRefreshToken проверяет, действителен ли refresh-токен, без его обновления
func (s *TokenService) CheckRefreshToken(refreshToken string) (bool, error) {
	return s.refreshTokenRepo.CheckToken(refreshToken)
}

// GetTokenInfo возвращает информацию о сроках действия текущих токенов
func (s *TokenService) GetTokenInfo(refreshToken string) (*TokenInfo, error) {
	// Находим refresh-токен в БД
	token, err := s.refreshTokenRepo.GetTokenByValue(refreshToken)
	if err != nil {
		return nil, err
	}

	// Вычисляем время истечения access-токена (примерно)
	// Это приблизительно, так как доступа к JWT токену нет
	accessTokenExpires := time.Now().Add(s.accessTokenExpiry)

	return &TokenInfo{
		AccessTokenExpires:  accessTokenExpires,
		RefreshTokenExpires: token.ExpiresAt,
	}, nil
}

// RevokeRefreshToken отзывает определенный refresh-токен
func (s *TokenService) RevokeRefreshToken(refreshToken string) error {
	// Помечаем токен как истекший вместо удаления
	if err := s.refreshTokenRepo.MarkTokenAsExpired(refreshToken); err != nil {
		log.Printf("[TokenService] Ошибка при отзыве refresh-токена: %v", err)
		return errors.New("ошибка отзыва токена")
	}

	log.Printf("[TokenService] Отозван refresh-токен")
	return nil
}

// RevokeAllUserTokens отзывает все refresh-токены пользователя
func (s *TokenService) RevokeAllUserTokens(userID uint) error {
	// Помечаем все refresh-токены пользователя как истекшие
	if err := s.refreshTokenRepo.MarkAllAsExpiredForUser(userID); err != nil {
		log.Printf("[TokenService] Ошибка при отзыве всех refresh-токенов пользователя ID=%d: %v", userID, err)
		return errors.New("ошибка отзыва токенов")
	}

	// Для обратной совместимости также инвалидируем JWT-токены
	if err := s.jwtService.InvalidateTokensForUser(userID); err != nil {
		log.Printf("[TokenService] Ошибка при инвалидации JWT-токенов пользователя ID=%d: %v", userID, err)
		// Не возвращаем ошибку, продолжаем (у нас уже отозваны refresh-токены)
	}

	log.Printf("[TokenService] Отозваны все токены пользователя ID=%d", userID)
	return nil
}

// CleanupExpiredTokens удаляет все истекшие refresh-токены
func (s *TokenService) CleanupExpiredTokens() error {
	count, err := s.refreshTokenRepo.CleanupExpiredTokens()
	if err != nil {
		log.Printf("[TokenService] Ошибка при очистке истекших токенов: %v", err)
		return err
	}

	// Для обратной совместимости также запускаем очистку инвалидированных JWT-токенов
	s.jwtService.CleanupInvalidatedUsers()

	log.Printf("[TokenService] Выполнена очистка %d истекших токенов", count)
	return nil
}

// GetUserActiveTokens возвращает все активные refresh-токены пользователя
func (s *TokenService) GetUserActiveTokens(userID uint) ([]entity.RefreshToken, error) {
	tokensPtr, err := s.refreshTokenRepo.GetActiveTokensForUser(userID)
	if err != nil {
		log.Printf("[TokenService] Ошибка при получении активных токенов пользователя ID=%d: %v", userID, err)
		return nil, errors.New("ошибка при получении списка сессий")
	}

	// Преобразуем []*entity.RefreshToken в []entity.RefreshToken
	tokens := make([]entity.RefreshToken, len(tokensPtr))
	for i, t := range tokensPtr {
		tokens[i] = *t
	}

	log.Printf("[TokenService] Получено %d активных токенов пользователя ID=%d", len(tokens), userID)
	return tokens, nil
}

// Служебные функции

// generateAccessToken генерирует новый access-токен (JWT)
func (s *TokenService) generateAccessToken(user *entity.User) (string, error) {
	return s.jwtService.GenerateToken(user)
}

// generateRefreshToken генерирует новый refresh-токен и сохраняет его в БД
func (s *TokenService) generateRefreshToken(userID uint, deviceID, ipAddress, userAgent string) (string, error) {
	// Генерируем случайный токен
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	tokenString := hex.EncodeToString(randomBytes)

	// Время истечения
	expiresAt := time.Now().Add(s.refreshTokenExpiry)

	// Создаем запись в БД
	token := entity.NewRefreshToken(userID, tokenString, deviceID, ipAddress, userAgent, expiresAt)

	// Сохраняем в БД
	_, err := s.refreshTokenRepo.CreateToken(token)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
