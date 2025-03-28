package repository

import (
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// InvalidTokenRepository определяет методы для работы с инвалидированными токенами
type InvalidTokenRepository interface {
	// AddInvalidToken добавляет запись об инвалидированном токене
	AddInvalidToken(userID uint, invalidationTime time.Time) error

	// RemoveInvalidToken удаляет запись об инвалидированном токене
	RemoveInvalidToken(userID uint) error

	// IsTokenInvalid проверяет, инвалидирован ли токен пользователя
	IsTokenInvalid(userID uint, tokenIssuedAt time.Time) (bool, error)

	// GetAllInvalidTokens возвращает все записи об инвалидированных токенах
	GetAllInvalidTokens() ([]entity.InvalidToken, error)

	// CleanupOldInvalidTokens удаляет устаревшие записи об инвалидированных токенах
	CleanupOldInvalidTokens(cutoffTime time.Time) error
}
