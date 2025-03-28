package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/trivia-api/pkg/auth"
	"github.com/yourusername/trivia-api/pkg/auth/manager"
)

// AuthMiddleware обеспечивает аутентификацию для защищенных маршрутов
type AuthMiddleware struct {
	jwtService   *auth.JWTService
	tokenService *auth.TokenService    // Устаревшее, для обратной совместимости
	tokenManager *manager.TokenManager // Новый менеджер токенов
}

// NewAuthMiddleware создает новый middleware аутентификации
// Устаревший метод, для обратной совместимости
func NewAuthMiddleware(jwtService *auth.JWTService, tokenService *auth.TokenService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:   jwtService,
		tokenService: tokenService,
	}
}

// NewAuthMiddlewareWithManager создает новый middleware с использованием TokenManager
func NewAuthMiddlewareWithManager(jwtService *auth.JWTService, tokenManager *manager.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:   jwtService,
		tokenManager: tokenManager,
	}
}

// WithTokenManager добавляет TokenManager к существующему middleware
func (m *AuthMiddleware) WithTokenManager(tokenManager *manager.TokenManager) *AuthMiddleware {
	m.tokenManager = tokenManager
	return m
}

// RequireAuth проверяет, аутентифицирован ли пользователь
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		var err error

		// Если доступен TokenManager, получаем токен из куки
		if m.tokenManager != nil {
			token, err = m.tokenManager.GetAccessTokenFromCookie(c.Request)
			if err != nil {
				// Если токен в куки не найден, проверяем заголовок для обратной совместимости
				authHeader := c.GetHeader("Authorization")
				if authHeader == "" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "error_type": "token_missing"})
					c.Abort()
					return
				}

				// Проверяем формат заголовка Bearer {token}
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}", "error_type": "token_format"})
					c.Abort()
					return
				}
				token = parts[1]
			}
		} else {
			// Для обратной совместимости используем только заголовок
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required", "error_type": "token_missing"})
				c.Abort()
				return
			}

			// Проверяем формат заголовка Bearer {token}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}", "error_type": "token_format"})
				c.Abort()
				return
			}
			token = parts[1]
		}

		// Проверяем токен
		claims, err := m.jwtService.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "error_type": "token_invalid"})
			c.Abort()
			return
		}

		// Устанавливаем ID пользователя в контекст
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		// Для администраторов добавляем флаг is_admin на основе проверки ID
		// (в будущих версиях это можно заменить на проверку claims.Role)
		if claims.UserID == 1 {
			c.Set("is_admin", true)
		}

		c.Next()
	}
}

// AdminOnly проверяет, является ли пользователь администратором
func (m *AuthMiddleware) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем, аутентифицирован ли пользователь
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Проверяем, является ли пользователь администратором
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			// Для обратной совместимости также проверяем по ID
			if userID.(uint) != 1 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin rights required"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// LogRequestInfo добавляет информацию о запросе в логи
func (m *AuthMiddleware) LogRequestInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Логируем информацию о запросе
		// Это может быть полезно для отладки
		// Например, логирование IP-адреса, User-Agent и т.д.

		// Передаем управление следующему middleware
		c.Next()
	}
}
