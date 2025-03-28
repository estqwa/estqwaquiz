package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config хранит все настройки приложения
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Auth      AuthConfig
	WebSocket WebSocketConfig
}

// ServerConfig содержит настройки HTTP сервера
type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
}

// DatabaseConfig содержит настройки подключения к PostgreSQL
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig содержит настройки подключения к Redis
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// JWTConfig содержит настройки JWT
type JWTConfig struct {
	Secret        string
	ExpirationHrs int
}

// AuthConfig содержит настройки аутентификации
type AuthConfig struct {
	SessionLimit         int
	RefreshTokenLifetime int
}

// WebSocketConfig содержит настройки WebSocket-подсистемы
type WebSocketConfig struct {
	Sharding ShardingConfig
	Buffers  BuffersConfig
	Priority PriorityConfig
	Ping     PingConfig
	Cluster  ClusterConfig
	Limits   LimitsConfig
}

// ShardingConfig содержит настройки шардирования
type ShardingConfig struct {
	Enabled              bool
	ShardCount           int
	MaxClientsPerShard   int
	BalancingInterval    int
	LoadThresholdPercent int
}

// BuffersConfig содержит настройки буферов
type BuffersConfig struct {
	ClientSendBuffer int
	BroadcastBuffer  int
	RegisterBuffer   int
	UnregisterBuffer int
}

// PriorityConfig содержит настройки приоритизации сообщений
type PriorityConfig struct {
	Enabled              bool
	HighPriorityBuffer   int
	NormalPriorityBuffer int
	LowPriorityBuffer    int
}

// PingConfig содержит настройки пингов
type PingConfig struct {
	Interval int
	Timeout  int
}

// ClusterConfig содержит настройки кластеризации
type ClusterConfig struct {
	Enabled          bool
	InstanceID       string
	BroadcastChannel string
	DirectChannel    string
	MetricsChannel   string
	MetricsInterval  int
}

// LimitsConfig содержит настройки ограничений
type LimitsConfig struct {
	MaxMessageSize      int
	WriteWait           int
	PongWait            int
	MaxConnectionsPerIP int
	CleanupInterval     int
}

// PostgresConnectionString формирует строку подключения к PostgreSQL
func (d *DatabaseConfig) PostgresConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// Load загружает конфигурацию из файла
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Проверка обязательных параметров
	if cfg.JWT.Secret == "" {
		log.Fatal("JWT secret is required")
	}

	if cfg.Database.Host == "" || cfg.Database.DBName == "" {
		log.Fatal("Database configuration is incomplete")
	}

	return &cfg, nil
}
