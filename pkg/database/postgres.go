package database

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// NewPostgresDB создает новое подключение к PostgreSQL
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Настройка пула соединений
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Максимальное число открытых соединений
	sqlDB.SetMaxOpenConns(25)

	// Максимальное число простаивающих соединений
	sqlDB.SetMaxIdleConns(10)

	// Максимальное время жизни соединения
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// MigrateDB создает необходимые таблицы в базе данных
func MigrateDB(db *gorm.DB) error {
	// Автоматическая миграция схемы БД
	err := db.AutoMigrate(
		&entity.User{},
		&entity.Quiz{},
		&entity.Question{},
		&entity.UserAnswer{},
		&entity.Result{},
		&entity.InvalidToken{},
		&entity.RefreshToken{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Получаем sql.DB для выполнения SQL-миграций
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for migrations: %w", err)
	}

	// Выполняем SQL-миграции для обновления полей таблицы refresh_tokens
	_, err = sqlDB.Exec(`
		-- Проверяем, есть ли уже колонка is_expired
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT FROM information_schema.columns 
				WHERE table_name = 'refresh_tokens' AND column_name = 'is_expired'
			) THEN
				-- Добавляем поле is_expired для маркировки токенов, вместо их удаления
				ALTER TABLE refresh_tokens ADD COLUMN is_expired BOOLEAN NOT NULL DEFAULT FALSE;

				-- Создаем индекс для быстрого поиска действительных токенов
				CREATE INDEX idx_refresh_tokens_not_expired ON refresh_tokens (user_id, is_expired) WHERE is_expired = FALSE;

				-- Обновляем имеющиеся токены, помечая их как действительные
				UPDATE refresh_tokens SET is_expired = FALSE;
			END IF;
		END
		$$;
	`)
	if err != nil {
		return fmt.Errorf("failed to execute refresh tokens migration: %w", err)
	}

	return nil
}

// GetSQLDB возвращает базовый *sql.DB из *gorm.DB
func GetSQLDB(gormDB *gorm.DB) (*sql.DB, error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB, nil
}
