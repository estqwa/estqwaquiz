package postgres

import (
	"database/sql"
	"log"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
)

// RefreshTokenRepo представляет PostgreSQL репозиторий для refresh-токенов
type RefreshTokenRepo struct {
	db *sql.DB
}

// NewRefreshTokenRepo создает новый PostgreSQL репозиторий для refresh-токенов
func NewRefreshTokenRepo(db *sql.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

// CreateToken сохраняет новый refresh-токен в базу данных
func (r *RefreshTokenRepo) CreateToken(refreshToken *entity.RefreshToken) (uint, error) {
	query := `
		INSERT INTO refresh_tokens (user_id, token, device_id, ip_address, user_agent, expires_at, created_at, is_expired) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		RETURNING id
	`

	var id uint
	err := r.db.QueryRow(
		query,
		refreshToken.UserID,
		refreshToken.Token,
		refreshToken.DeviceID,
		refreshToken.IPAddress,
		refreshToken.UserAgent,
		refreshToken.ExpiresAt,
		refreshToken.CreatedAt,
		false, // По умолчанию токен действителен
	).Scan(&id)

	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при создании refresh-токена: %v", err)
		return 0, err
	}

	log.Printf("[RefreshTokenRepo] Создан новый refresh-токен ID=%d для пользователя ID=%d", id, refreshToken.UserID)
	return id, nil
}

// GetTokenByValue находит refresh-токен по его значению
func (r *RefreshTokenRepo) GetTokenByValue(token string) (*entity.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, device_id, ip_address, user_agent, expires_at, created_at, is_expired
		FROM refresh_tokens 
		WHERE token = $1
	`

	refreshToken := &entity.RefreshToken{}
	err := r.db.QueryRow(query, token).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.Token,
		&refreshToken.DeviceID,
		&refreshToken.IPAddress,
		&refreshToken.UserAgent,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&refreshToken.IsExpired,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[RefreshTokenRepo] Refresh-токен не найден: %s", token)
			return nil, repository.ErrNotFound
		}
		log.Printf("[RefreshTokenRepo] Ошибка при поиске refresh-токена: %v", err)
		return nil, err
	}

	// Проверяем, не помечен ли токен как истекший
	if refreshToken.IsExpired {
		log.Printf("[RefreshTokenRepo] Refresh-токен помечен как истекший: %s", token)
		return nil, repository.ErrExpiredToken
	}

	return refreshToken, nil
}

// CheckToken проверяет действительность refresh-токена без получения полной информации
func (r *RefreshTokenRepo) CheckToken(token string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM refresh_tokens 
			WHERE token = $1 AND expires_at > $2 AND NOT is_expired
		)
	`

	var exists bool
	err := r.db.QueryRow(query, token, time.Now()).Scan(&exists)

	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при проверке refresh-токена: %v", err)
		return false, err
	}

	return exists, nil
}

// MarkTokenAsExpired помечает refresh-токен как истекший вместо его удаления
func (r *RefreshTokenRepo) MarkTokenAsExpired(token string) error {
	query := `UPDATE refresh_tokens SET is_expired = TRUE WHERE token = $1`

	result, err := r.db.Exec(query, token)
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при маркировке refresh-токена как истекшего: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[RefreshTokenRepo] Помечено как истекшие %d refresh-токенов", rowsAffected)
	return nil
}

// DeleteToken физически удаляет refresh-токен (используется только для критических операций)
func (r *RefreshTokenRepo) DeleteToken(token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`

	result, err := r.db.Exec(query, token)
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при удалении refresh-токена: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[RefreshTokenRepo] Удалено %d refresh-токенов", rowsAffected)
	return nil
}

// MarkAllAsExpiredForUser помечает все refresh-токены пользователя как истекшие
func (r *RefreshTokenRepo) MarkAllAsExpiredForUser(userID uint) error {
	query := `UPDATE refresh_tokens SET is_expired = TRUE WHERE user_id = $1`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при маркировке всех refresh-токенов пользователя ID=%d как истекших: %v", userID, err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[RefreshTokenRepo] Помечено как истекшие %d refresh-токенов для пользователя ID=%d", rowsAffected, userID)
	return nil
}

// CleanupExpiredTokens физически удаляет истекшие токены, помеченные как expired
func (r *RefreshTokenRepo) CleanupExpiredTokens() (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE (expires_at < $1 OR is_expired = TRUE) AND created_at < $2`

	// Удаляем токены, которые истекли или помечены как expired И были созданы более 7 дней назад
	// Это позволяет сохранять историю токенов для аудита и отладки на некоторое время
	result, err := r.db.Exec(query, time.Now(), time.Now().AddDate(0, 0, -7))
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при очистке просроченных refresh-токенов: %v", err)
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[RefreshTokenRepo] Удалено %d просроченных refresh-токенов", rowsAffected)
	return rowsAffected, nil
}

// GetActiveTokensForUser получает все активные refresh-токены для указанного пользователя
func (r *RefreshTokenRepo) GetActiveTokensForUser(userID uint) ([]*entity.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, device_id, ip_address, user_agent, expires_at, created_at, is_expired
		FROM refresh_tokens 
		WHERE user_id = $1 AND expires_at > $2 AND NOT is_expired
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID, time.Now())
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при получении активных refresh-токенов пользователя ID=%d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	var tokens []*entity.RefreshToken
	for rows.Next() {
		token := &entity.RefreshToken{}
		err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Token,
			&token.DeviceID,
			&token.IPAddress,
			&token.UserAgent,
			&token.ExpiresAt,
			&token.CreatedAt,
			&token.IsExpired,
		)
		if err != nil {
			log.Printf("[RefreshTokenRepo] Ошибка при сканировании строки refresh-токена: %v", err)
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при итерации по строкам refresh-токенов: %v", err)
		return nil, err
	}

	log.Printf("[RefreshTokenRepo] Получено %d активных refresh-токенов для пользователя ID=%d", len(tokens), userID)
	return tokens, nil
}

// CountTokensForUser подсчитывает количество активных refresh-токенов для указанного пользователя
func (r *RefreshTokenRepo) CountTokensForUser(userID uint) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM refresh_tokens 
		WHERE user_id = $1 AND expires_at > $2 AND NOT is_expired
	`

	var count int
	err := r.db.QueryRow(query, userID, time.Now()).Scan(&count)
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при подсчете активных refresh-токенов пользователя ID=%d: %v", userID, err)
		return 0, err
	}

	return count, nil
}

// MarkOldestAsExpiredForUser помечает самые старые refresh-токены пользователя как истекшие, оставляя только указанное количество
func (r *RefreshTokenRepo) MarkOldestAsExpiredForUser(userID uint, limit int) error {
	query := `
		UPDATE refresh_tokens
		SET is_expired = TRUE
		WHERE id IN (
			SELECT id
			FROM refresh_tokens
			WHERE user_id = $1 AND NOT is_expired
			ORDER BY created_at ASC
			OFFSET $2
		)
	`

	result, err := r.db.Exec(query, userID, limit)
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при маркировке старых refresh-токенов пользователя ID=%d как истекших: %v", userID, err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[RefreshTokenRepo] Помечено как истекшие %d старых refresh-токенов для пользователя ID=%d", rowsAffected, userID)
	return nil
}

// GetTokenByID находит refresh-токен по его ID
func (r *RefreshTokenRepo) GetTokenByID(id uint) (*entity.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, device_id, ip_address, user_agent, 
		       expires_at, created_at, is_expired, revoked_at, reason
		FROM refresh_tokens 
		WHERE id = $1
	`

	var refreshToken entity.RefreshToken
	var revokedAt sql.NullTime
	var reason sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.Token,
		&refreshToken.DeviceID,
		&refreshToken.IPAddress,
		&refreshToken.UserAgent,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&refreshToken.IsExpired,
		&revokedAt,
		&reason,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[RefreshTokenRepo] Refresh-токен с ID=%d не найден", id)
			return nil, nil
		}
		log.Printf("[RefreshTokenRepo] Ошибка при получении refresh-токена по ID=%d: %v", id, err)
		return nil, err
	}

	if revokedAt.Valid {
		refreshToken.RevokedAt = &revokedAt.Time
	}
	if reason.Valid {
		refreshToken.Reason = reason.String
	}

	return &refreshToken, nil
}

// MarkTokenAsExpiredByID помечает токен как истекший по его ID
func (r *RefreshTokenRepo) MarkTokenAsExpiredByID(id uint) error {
	query := `
		UPDATE refresh_tokens
		SET is_expired = true, revoked_at = $1, reason = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(query, time.Now(), "Manually revoked", id)
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при пометке токена ID=%d как истекшего: %v", id, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[RefreshTokenRepo] Ошибка при получении количества затронутых строк: %v", err)
		return err
	}

	if rowsAffected == 0 {
		log.Printf("[RefreshTokenRepo] Токен с ID=%d не найден при попытке пометить его как истекший", id)
		return nil
	}

	log.Printf("[RefreshTokenRepo] Токен ID=%d помечен как истекший", id)
	return nil
}
