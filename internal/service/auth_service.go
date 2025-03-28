package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
	"github.com/yourusername/trivia-api/pkg/auth"
	"github.com/yourusername/trivia-api/pkg/auth/manager"
)

// AuthService предоставляет методы для аутентификации пользователей
type AuthService struct {
	userRepo         repository.UserRepository
	jwtService       *auth.JWTService
	tokenService     *auth.TokenService                // Устаревшее поле для обратной совместимости
	tokenManager     *manager.TokenManager             // Новое поле для работы с токенами
	refreshTokenRepo repository.RefreshTokenRepository // Добавляем прямой доступ к репозиторию refresh-токенов
}

// NewAuthService создает новый сервис аутентификации
func NewAuthService(
	userRepo repository.UserRepository,
	jwtService *auth.JWTService,
	tokenService *auth.TokenService,
	refreshTokenRepo repository.RefreshTokenRepository,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		jwtService:       jwtService,
		tokenService:     tokenService,
		refreshTokenRepo: refreshTokenRepo,
	}
}

// WithTokenManager устанавливает TokenManager для AuthService
func (s *AuthService) WithTokenManager(tokenManager *manager.TokenManager) *AuthService {
	s.tokenManager = tokenManager
	return s
}

// RegisterUser регистрирует нового пользователя
func (s *AuthService) RegisterUser(username, email, password string) (*entity.User, error) {
	existingUser, _ := s.userRepo.GetByEmail(email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	existingUser, _ = s.userRepo.GetByUsername(username)
	if existingUser != nil {
		return nil, errors.New("username already taken")
	}

	user := &entity.User{
		Username:    username,
		Email:       email,
		Password:    password, // ✅ Передаем обычный пароль, GORM сам вызовет BeforeSave()
		GamesPlayed: 0,
		TotalScore:  0,
	}

	// ✅ Сохраняем пользователя в БД (GORM вызовет BeforeSave)
	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// AuthResponse содержит данные для ответа на запрос авторизации
type AuthResponse struct {
	User         *entity.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// LoginUser аутентифицирует пользователя и возвращает пару токенов
func (s *AuthService) LoginUser(email, password, deviceID, ipAddress, userAgent string) (*AuthResponse, error) {
	// Ищем пользователя по email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	log.Println("Вход пользователя:", user.Email)
	log.Println("Хеш пароля в БД:", user.Password)
	log.Println("Введенный пароль:", password)

	// Проверяем пароль
	if !user.CheckPassword(password) {
		log.Println("Ошибка аутентификации: неверный пароль для пользователя", user.Email)
		return nil, errors.New("invalid email or password")
	}

	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		// Генерируем пару токенов через TokenManager
		tokenResp, err := s.tokenManager.GenerateTokenPair(user.ID, deviceID, ipAddress, userAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tokens: %w", err)
		}

		// Получаем refresh токен
		refreshToken, err := s.GetRefreshTokenByUserID(user.ID)
		if err != nil {
			log.Printf("Ошибка получения refresh токена для пользователя %d: %v", user.ID, err)
			// Продолжаем без refresh токена
		}

		// Формируем ответ
		response := &AuthResponse{
			User:        user,
			AccessToken: tokenResp.AccessToken,
		}

		// Добавляем refresh токен если нужно (для обратной совместимости)
		if refreshToken != nil {
			response.RefreshToken = refreshToken.Token
		}

		return response, nil
	}

	// Для обратной совместимости с TokenService
	if s.tokenService != nil {
		// Генерируем пару токенов
		tokenPair, err := s.tokenService.GenerateTokenPair(user.ID, deviceID, ipAddress, userAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tokens: %w", err)
		}

		return &AuthResponse{
			User:         user,
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
		}, nil
	}

	// Для обратной совместимости, если tokenService не инициализирован
	// Генерируем только JWT токен
	accessToken, err := s.jwtService.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: accessToken,
	}, nil
}

// RefreshTokens обновляет пару токенов по refresh-токену
func (s *AuthService) RefreshTokens(refreshToken, deviceID, ipAddress, userAgent string) (*AuthResponse, error) {
	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		// CSRF токен получаем из заголовка запроса в хэндлере, здесь достаточно пустой строки
		tokenResp, err := s.tokenManager.RefreshTokens(refreshToken, "", deviceID, ipAddress, userAgent)
		if err != nil {
			return nil, err
		}

		// Получаем данные пользователя
		user, err := s.userRepo.GetByID(tokenResp.UserID)
		if err != nil {
			return nil, errors.New("failed to get user info")
		}

		return &AuthResponse{
			User:        user,
			AccessToken: tokenResp.AccessToken,
		}, nil
	}

	// Для обратной совместимости с TokenService
	if s.tokenService == nil {
		return nil, errors.New("token service not available")
	}

	tokenPair, err := s.tokenService.RefreshTokens(refreshToken, deviceID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	// Получаем данные из нового refresh-токена напрямую из репозитория
	token, err := s.refreshTokenRepo.GetTokenByValue(tokenPair.RefreshToken)
	if err != nil {
		return nil, errors.New("failed to get user info")
	}

	user, err := s.userRepo.GetByID(token.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

// GetUserByID возвращает пользователя по ID
func (s *AuthService) GetUserByID(userID uint) (*entity.User, error) {
	return s.userRepo.GetByID(userID)
}

// UpdateUserProfile обновляет профиль пользователя
func (s *AuthService) UpdateUserProfile(userID uint, username, profilePicture string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	// Если имя пользователя изменилось, проверяем, что оно уникально
	if username != user.Username {
		existingUser, _ := s.userRepo.GetByUsername(username)
		if existingUser != nil {
			return errors.New("username already taken")
		}
	}

	// Используем безопасный метод обновления профиля без изменения пароля
	updates := map[string]interface{}{
		"username":        username,
		"profile_picture": profilePicture,
	}

	return s.userRepo.UpdateProfile(userID, updates)
}

// ChangePassword изменяет пароль пользователя и инвалидирует все токены
func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// Получаем пользователя для проверки старого пароля
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	// Проверяем, что старый пароль верный
	if !user.CheckPassword(oldPassword) {
		return errors.New("incorrect old password")
	}

	// Обновляем пароль с использованием безопасного метода
	// UserRepo.UpdatePassword выполняет хеширование и использует прямой SQL-запрос
	// для обхода хука BeforeSave и предотвращения двойного хеширования
	if err := s.userRepo.UpdatePassword(userID, newPassword); err != nil {
		return err
	}

	// Инвалидируем все токены пользователя
	return s.LogoutAllDevices(userID)
}

// LogoutUser выполняет выход пользователя
func (s *AuthService) LogoutUser(userID uint, refreshToken string) error {
	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		return s.tokenManager.RevokeRefreshToken(refreshToken)
	}

	// Для обратной совместимости с TokenService
	if s.tokenService != nil && refreshToken != "" {
		// Отзываем только конкретный refresh-токен
		return s.tokenService.RevokeRefreshToken(refreshToken)
	}

	// Для обратной совместимости, если tokenService не инициализирован
	// или refresh-токен не предоставлен, инвалидируем все токены
	return s.jwtService.InvalidateTokensForUser(userID)
}

// LogoutAllDevices выполняет выход пользователя со всех устройств
func (s *AuthService) LogoutAllDevices(userID uint) error {
	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		return s.tokenManager.RevokeAllUserTokens(userID)
	}

	// Для обратной совместимости с TokenService
	if s.tokenService != nil {
		// Отзываем все refresh-токены пользователя
		return s.tokenService.RevokeAllUserTokens(userID)
	}

	// Для обратной совместимости, если tokenService не инициализирован
	return s.jwtService.InvalidateTokensForUser(userID)
}

// ResetUserTokenInvalidation сбрасывает все инвалидации токенов для пользователя
// Используется для решения проблем с аутентификацией старых аккаунтов
func (s *AuthService) ResetUserTokenInvalidation(userID uint) {
	log.Printf("DEBUG: Resetting token invalidation for user ID=%d", userID)
	s.jwtService.ResetInvalidationForUser(userID)
}

// GetUserActiveSessions возвращает все активные сессии пользователя
func (s *AuthService) GetUserActiveSessions(userID uint) ([]entity.RefreshToken, error) {
	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		// У TokenManager нет прямого метода, используем репозиторий
		tokensPtr, err := s.refreshTokenRepo.GetActiveTokensForUser(userID)
		if err != nil {
			return nil, err
		}

		// Преобразуем []*entity.RefreshToken в []entity.RefreshToken
		tokens := make([]entity.RefreshToken, len(tokensPtr))
		for i, t := range tokensPtr {
			tokens[i] = *t
		}

		return tokens, nil
	}

	if s.tokenService == nil {
		return []entity.RefreshToken{}, nil
	}

	return s.tokenService.GetUserActiveTokens(userID)
}

// TokenInfo содержит информацию о сроке действия токенов
type TokenInfo struct {
	AccessTokenExpires  time.Time `json:"access_token_expires"`
	RefreshTokenExpires time.Time `json:"refresh_token_expires"`
}

// CheckRefreshToken проверяет валидность refresh-токена без обновления
func (s *AuthService) CheckRefreshToken(refreshToken string) (bool, error) {
	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		// Используем репозиторий напрямую
		return s.refreshTokenRepo.CheckToken(refreshToken)
	}

	return s.tokenService.CheckRefreshToken(refreshToken)
}

// GetTokenInfo возвращает информацию о сроке действия токенов
func (s *AuthService) GetTokenInfo(refreshToken string) (*TokenInfo, error) {
	// Если доступен TokenManager, используем его
	if s.tokenManager != nil {
		managerInfo, err := s.tokenManager.GetTokenInfo(refreshToken)
		if err != nil {
			return nil, err
		}

		return &TokenInfo{
			AccessTokenExpires:  managerInfo.AccessTokenExpires,
			RefreshTokenExpires: managerInfo.RefreshTokenExpires,
		}, nil
	}

	info, err := s.tokenService.GetTokenInfo(refreshToken)
	if err != nil {
		return nil, err
	}

	return &TokenInfo{
		AccessTokenExpires:  info.AccessTokenExpires,
		RefreshTokenExpires: info.RefreshTokenExpires,
	}, nil
}

// DebugToken анализирует JWT токен без проверки подписи
// для диагностических целей
func (s *AuthService) DebugToken(tokenString string) map[string]interface{} {
	return s.jwtService.DebugToken(tokenString)
}

// GetUserByEmail возвращает пользователя по Email
func (s *AuthService) GetUserByEmail(email string) (*entity.User, error) {
	return s.userRepo.GetByEmail(email)
}

// AdminResetPassword сбрасывает пароль пользователя администратором
// Не требует проверки старого пароля и инвалидирует все токены пользователя
func (s *AuthService) AdminResetPassword(userID uint, newPassword string) error {
	// Обновляем пароль с использованием безопасного метода
	// UserRepo.UpdatePassword выполняет хеширование и использует прямой SQL-запрос
	// для обхода хука BeforeSave и предотвращения двойного хеширования
	if err := s.userRepo.UpdatePassword(userID, newPassword); err != nil {
		return err
	}

	// Инвалидируем все токены пользователя
	return s.LogoutAllDevices(userID)
}

// GetRefreshTokenByUserID получает активный refresh токен пользователя
func (s *AuthService) GetRefreshTokenByUserID(userID uint) (*entity.RefreshToken, error) {
	tokens, err := s.refreshTokenRepo.GetActiveTokensForUser(userID)
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, errors.New("no active refresh tokens found")
	}

	// Возвращаем первый активный токен
	return tokens[0], nil
}

// AuthenticateUser проверяет учетные данные пользователя без создания токенов
func (s *AuthService) AuthenticateUser(email, password string) (*entity.User, error) {
	// Получаем пользователя по email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		log.Printf("[AuthService] Пользователь с email %s не найден: %v", email, err)
		return nil, errors.New("неверные учетные данные")
	}

	// Проверяем пароль
	if !user.CheckPassword(password) {
		log.Printf("[AuthService] Неверный пароль для пользователя с email %s", email)
		return nil, errors.New("неверные учетные данные")
	}

	return user, nil
}

// IsSessionOwnedByUser проверяет, принадлежит ли сессия пользователю
func (s *AuthService) IsSessionOwnedByUser(userID, sessionID uint) (bool, error) {
	if s.refreshTokenRepo == nil {
		return false, errors.New("refresh token repository not available")
	}

	// Получаем токен по ID
	token, err := s.refreshTokenRepo.GetTokenByID(sessionID)
	if err != nil {
		if err == repository.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	// Проверяем, что токен принадлежит пользователю
	return token.UserID == userID, nil
}

// RevokeSession отзывает отдельную сессию по ID
func (s *AuthService) RevokeSession(sessionID uint) error {
	if s.refreshTokenRepo == nil {
		return errors.New("refresh token repository not available")
	}

	return s.refreshTokenRepo.MarkTokenAsExpiredByID(sessionID)
}

// GetRefreshTokenByID получает refresh-токен по его ID
func (s *AuthService) GetRefreshTokenByID(tokenID uint) (*entity.RefreshToken, error) {
	return s.refreshTokenRepo.GetTokenByID(tokenID)
}

// RevokeSessionByID отзывает сессию по её ID с указанием причины
func (s *AuthService) RevokeSessionByID(sessionID uint, reason string) error {
	// Получаем сессию по ID
	token, err := s.refreshTokenRepo.GetTokenByID(sessionID)
	if err != nil {
		return fmt.Errorf("не удалось найти сессию: %w", err)
	}

	// Устанавливаем время отзыва и причину
	now := time.Now()
	token.RevokedAt = &now
	token.Reason = reason
	token.IsExpired = true

	// Сохраняем изменения через метод MarkTokenAsExpiredByID
	err = s.refreshTokenRepo.MarkTokenAsExpiredByID(sessionID)
	if err != nil {
		return fmt.Errorf("не удалось отозвать сессию: %w", err)
	}

	return nil
}

// RevokeAllUserSessions отзывает все сессии пользователя с указанием причины
func (s *AuthService) RevokeAllUserSessions(userID uint, reason string) error {
	// Получаем все активные сессии пользователя
	tokens, err := s.refreshTokenRepo.GetActiveTokensForUser(userID)
	if err != nil {
		return fmt.Errorf("не удалось получить активные сессии: %w", err)
	}

	// Отзываем каждую сессию с указанием причины
	for _, token := range tokens {
		now := time.Now()
		token.RevokedAt = &now
		token.Reason = reason
		token.IsExpired = true

		err = s.refreshTokenRepo.MarkTokenAsExpiredByID(token.ID)
		if err != nil {
			log.Printf("Ошибка при отзыве сессии ID=%d: %v", token.ID, err)
			// Продолжаем отзыв других сессий
		}
	}

	return nil
}

// GetActiveSessionsWithDetails возвращает детализированную информацию о сессиях пользователя
func (s *AuthService) GetActiveSessionsWithDetails(userID uint) ([]map[string]interface{}, error) {
	tokens, err := s.refreshTokenRepo.GetActiveTokensForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить активные сессии: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(tokens))
	for _, token := range tokens {
		// Создаем детализированную информацию о сессии
		sessionInfo := map[string]interface{}{
			"id":         token.ID,
			"device_id":  token.DeviceID,
			"ip_address": token.IPAddress,
			"user_agent": token.UserAgent,
			"created_at": token.CreatedAt,
			"expires_at": token.ExpiresAt,
			"is_expired": token.IsExpired,
			"last_used":  token.CreatedAt, // По умолчанию время создания
		}

		// Добавляем информацию об отзыве, если сессия отозвана
		if token.RevokedAt != nil {
			sessionInfo["revoked_at"] = token.RevokedAt
			sessionInfo["reason"] = token.Reason
		}

		result = append(result, sessionInfo)
	}

	return result, nil
}

// GenerateWsTicket генерирует краткоживущий токен для WebSocket подключения
func (s *AuthService) GenerateWsTicket(userID uint, email string) (string, error) {
	// Используем JWTService для генерации WS-тикета
	return s.jwtService.GenerateWSTicket(userID, email)
}
