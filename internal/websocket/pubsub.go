package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// PubSubProvider определяет интерфейс для провайдеров публикации/подписки
type PubSubProvider interface {
	// Publish публикует сообщение в указанный канал
	Publish(channel string, message []byte) error

	// Subscribe подписывается на указанный канал и возвращает канал для сообщений
	Subscribe(ctx context.Context, channel string) (<-chan []byte, error)

	// Close закрывает все соединения и освобождает ресурсы
	Close() error
}

// ClusterMessage представляет сообщение, передаваемое между экземплярами Hub
type ClusterMessage struct {
	// MessageType определяет тип сообщения кластера
	// broadcast - широковещательное сообщение для всех клиентов
	// direct - сообщение для конкретного пользователя
	// metrics - обновление метрик кластера
	MessageType string `json:"type"`

	// RecipientID содержит ID получателя для direct-сообщений
	RecipientID string `json:"recipient_id,omitempty"`

	// InstanceID содержит ID отправителя для избежания дублирования
	InstanceID string `json:"instance_id"`

	// Payload содержит данные сообщения
	Payload json.RawMessage `json:"payload"`

	// Timestamp содержит время создания сообщения
	Timestamp time.Time `json:"timestamp"`
}

// NoOpPubSub реализует PubSubProvider для одиночного режима работы
// Этот провайдер не выполняет реальных действий и используется, когда
// горизонтальное масштабирование отключено
type NoOpPubSub struct{}

// Publish реализует метод PubSubProvider.Publish для NoOpPubSub
func (p *NoOpPubSub) Publish(channel string, message []byte) error {
	// Ничего не делаем в одиночном режиме
	return nil
}

// Subscribe реализует метод PubSubProvider.Subscribe для NoOpPubSub
func (p *NoOpPubSub) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	// Возвращаем пустой канал, который никогда не получит сообщения
	msgCh := make(chan []byte)
	go func() {
		<-ctx.Done()
		close(msgCh)
	}()
	return msgCh, nil
}

// Close реализует метод PubSubProvider.Close для NoOpPubSub
func (p *NoOpPubSub) Close() error {
	return nil
}

// ClusterConfig содержит настройки кластера Hub
type ClusterConfig struct {
	// Включение режима кластера
	Enabled bool

	// Уникальный ID этого экземпляра Hub
	InstanceID string

	// Провайдер публикации/подписки
	Provider PubSubProvider

	// Канал для широковещательных сообщений
	BroadcastChannel string

	// Канал для прямых сообщений
	DirectChannel string

	// Канал для обмена метриками
	MetricsChannel string

	// Интервал обновления метрик
	MetricsInterval time.Duration
}

// DefaultClusterConfig возвращает конфигурацию кластера по умолчанию
func DefaultClusterConfig() ClusterConfig {
	return ClusterConfig{
		Enabled:          false,
		InstanceID:       generateInstanceID(),
		Provider:         &NoOpPubSub{},
		BroadcastChannel: "trivia_ws_broadcast",
		DirectChannel:    "trivia_ws_direct",
		MetricsChannel:   "trivia_ws_metrics",
		MetricsInterval:  30 * time.Second,
	}
}

// generateInstanceID создает уникальный ID для экземпляра Hub
func generateInstanceID() string {
	return "instance_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString генерирует случайную строку указанной длины
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Добавляем минимальную задержку для лучшей случайности
	}
	return string(result)
}

// ClusterHub управляет коммуникацией между экземплярами Hub
type ClusterHub struct {
	config  ClusterConfig
	parent  interface{} // Ссылка на родительский хаб, меняем *ShardedHub на interface{}
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	metrics *HubMetrics
}

// NewClusterHub создает новый ClusterHub
func NewClusterHub(parent interface{}, config ClusterConfig, metrics *HubMetrics) *ClusterHub {
	ctx, cancel := context.WithCancel(context.Background())
	return &ClusterHub{
		config:  config,
		parent:  parent,
		ctx:     ctx,
		cancel:  cancel,
		metrics: metrics,
	}
}

// Start запускает обработку сообщений кластера
func (ch *ClusterHub) Start() error {
	if !ch.config.Enabled {
		log.Println("ClusterHub: кластерный режим отключен, работаем в автономном режиме")
		return nil
	}

	log.Printf("ClusterHub: запуск кластерного режима, ID экземпляра: %s", ch.config.InstanceID)

	// Подписываемся на широковещательные сообщения
	ch.wg.Add(1)
	go func() {
		defer ch.wg.Done()
		ch.handleBroadcastMessages()
	}()

	// Подписываемся на прямые сообщения
	ch.wg.Add(1)
	go func() {
		defer ch.wg.Done()
		ch.handleDirectMessages()
	}()

	// Запускаем периодическую отправку метрик
	ch.wg.Add(1)
	go func() {
		defer ch.wg.Done()
		ch.publishMetrics()
	}()

	return nil
}

// Stop останавливает обработку сообщений кластера
func (ch *ClusterHub) Stop() {
	if !ch.config.Enabled {
		return
	}

	log.Println("ClusterHub: остановка кластерного режима")
	ch.cancel()
	ch.wg.Wait()
}

// BroadcastToCluster отправляет широковещательное сообщение всем экземплярам Hub
func (ch *ClusterHub) BroadcastToCluster(payload []byte) error {
	if !ch.config.Enabled {
		return nil
	}

	msg := ClusterMessage{
		MessageType: "broadcast",
		InstanceID:  ch.config.InstanceID,
		Payload:     payload,
		Timestamp:   time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return ch.config.Provider.Publish(ch.config.BroadcastChannel, data)
}

// SendToUserInCluster отправляет сообщение конкретному пользователю через кластер
func (ch *ClusterHub) SendToUserInCluster(userID string, payload []byte) error {
	if !ch.config.Enabled {
		return nil
	}

	msg := ClusterMessage{
		MessageType: "direct",
		RecipientID: userID,
		InstanceID:  ch.config.InstanceID,
		Payload:     payload,
		Timestamp:   time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return ch.config.Provider.Publish(ch.config.DirectChannel, data)
}

// handleBroadcastMessages обрабатывает входящие широковещательные сообщения
func (ch *ClusterHub) handleBroadcastMessages() {
	broadcastCh, err := ch.config.Provider.Subscribe(ch.ctx, ch.config.BroadcastChannel)
	if err != nil {
		log.Printf("ClusterHub: ошибка подписки на канал %s: %v", ch.config.BroadcastChannel, err)
		return
	}

	log.Printf("ClusterHub: начата обработка широковещательных сообщений")

	for {
		select {
		case <-ch.ctx.Done():
			return
		case data, ok := <-broadcastCh:
			if !ok {
				log.Println("ClusterHub: канал широковещательных сообщений закрыт")
				return
			}

			var msg ClusterMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("ClusterHub: ошибка разбора широковещательного сообщения: %v", err)
				continue
			}

			// Игнорируем сообщения от этого же экземпляра
			if msg.InstanceID == ch.config.InstanceID {
				continue
			}

			// Передаем сообщение в Hub для рассылки
			if ch.parent != nil {
				log.Printf("ClusterHub: получено широковещательное сообщение от %s", msg.InstanceID)
				if sh, ok := ch.parent.(*ShardedHub); ok {
					sh.BroadcastBytes([]byte(msg.Payload))
				}
			}
		}
	}
}

// handleDirectMessages обрабатывает входящие прямые сообщения
func (ch *ClusterHub) handleDirectMessages() {
	directCh, err := ch.config.Provider.Subscribe(ch.ctx, ch.config.DirectChannel)
	if err != nil {
		log.Printf("ClusterHub: ошибка подписки на канал %s: %v", ch.config.DirectChannel, err)
		return
	}

	log.Printf("ClusterHub: начата обработка прямых сообщений")

	for {
		select {
		case <-ch.ctx.Done():
			return
		case data, ok := <-directCh:
			if !ok {
				log.Println("ClusterHub: канал прямых сообщений закрыт")
				return
			}

			var msg ClusterMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("ClusterHub: ошибка разбора прямого сообщения: %v", err)
				continue
			}

			// Игнорируем сообщения от этого же экземпляра
			if msg.InstanceID == ch.config.InstanceID {
				continue
			}

			// Передаем сообщение в Hub для доставки пользователю
			if ch.parent != nil && msg.RecipientID != "" {
				log.Printf("ClusterHub: получено прямое сообщение от %s для пользователя %s",
					msg.InstanceID, msg.RecipientID)
				if sh, ok := ch.parent.(*ShardedHub); ok {
					sh.SendToUser(msg.RecipientID, []byte(msg.Payload))
				}
			}
		}
	}
}

// publishMetrics периодически публикует метрики экземпляра Hub
func (ch *ClusterHub) publishMetrics() {
	ticker := time.NewTicker(ch.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ch.ctx.Done():
			return
		case <-ticker.C:
			if ch.metrics == nil {
				continue
			}

			// Получаем метрики
			metrics := ch.metrics.GetBasicMetrics()

			// Добавляем информацию об экземпляре
			metrics["instance_id"] = ch.config.InstanceID
			metrics["timestamp"] = time.Now().Format(time.RFC3339)

			// Сериализуем метрики
			data, err := json.Marshal(metrics)
			if err != nil {
				log.Printf("ClusterHub: ошибка сериализации метрик: %v", err)
				continue
			}

			// Создаем сообщение кластера
			msg := ClusterMessage{
				MessageType: "metrics",
				InstanceID:  ch.config.InstanceID,
				Payload:     data,
				Timestamp:   time.Now(),
			}

			// Сериализуем сообщение
			msgData, err := json.Marshal(msg)
			if err != nil {
				log.Printf("ClusterHub: ошибка сериализации сообщения с метриками: %v", err)
				continue
			}

			// Публикуем метрики
			if err := ch.config.Provider.Publish(ch.config.MetricsChannel, msgData); err != nil {
				log.Printf("ClusterHub: ошибка публикации метрик: %v", err)
			}
		}
	}
}

// RedisConfig содержит настройки для подключения к Redis
type RedisConfig struct {
	Addresses   []string
	Password    string
	DB          int
	MasterName  string
	UseCluster  bool
	UseSentinel bool
	MaxRetries  int
}

// =====================================================================
// РЕАЛИЗАЦИЯ REDIS PUB/SUB
// =====================================================================

// RedisPubSub реализует PubSubProvider с использованием Redis
type RedisPubSub struct {
	client        *redis.Client
	config        RedisConfig
	ctx           context.Context
	cancel        context.CancelFunc
	subscriptions sync.Map
	retryInterval time.Duration
	maxRetries    int
	mu            sync.Mutex
}

// NewRedisPubSub создает новый провайдер Redis Pub/Sub
func NewRedisPubSub(config RedisConfig) (*RedisPubSub, error) {
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("RedisPubSub: создание провайдера Redis Pub/Sub")

	var client *redis.Client

	// Создаем клиент Redis в зависимости от настроек
	if config.UseCluster {
		// Кластерный режим не поддерживается в этой реализации
		log.Printf("RedisPubSub: кластерный режим не поддерживается, используйте стандартный режим")
		cancel()
		return nil, fmt.Errorf("кластерный режим Redis не поддерживается")
	} else if config.UseSentinel {
		// Sentinel режим не поддерживается в этой реализации
		log.Printf("RedisPubSub: режим Sentinel не поддерживается, используйте стандартный режим")
		cancel()
		return nil, fmt.Errorf("режим Sentinel Redis не поддерживается")
	} else {
		// Обычный режим
		addr := ""
		if len(config.Addresses) > 0 {
			addr = config.Addresses[0]
		} else {
			addr = "localhost:6379"
		}

		client = redis.NewClient(&redis.Options{
			Addr:       addr,
			Password:   config.Password,
			DB:         config.DB,
			MaxRetries: config.MaxRetries,
		})
	}

	// Проверяем подключение
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("RedisPubSub: ошибка подключения к Redis: %v", err)
		cancel()
		return nil, fmt.Errorf("ошибка подключения к Redis: %w", err)
	}

	log.Printf("RedisPubSub: успешное подключение к Redis")

	return &RedisPubSub{
		client:        client,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		retryInterval: 1 * time.Second,
		maxRetries:    config.MaxRetries,
	}, nil
}

// Publish публикует сообщение в указанный канал
func (p *RedisPubSub) Publish(channel string, message []byte) error {
	var err error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		// Публикуем сообщение
		err = p.client.Publish(p.ctx, channel, message).Err()
		if err == nil {
			return nil
		}

		// Если ошибка не связана с подключением, прекращаем попытки
		if !isRedisConnError(err) {
			return err
		}

		// Логируем ошибку и делаем повторную попытку
		log.Printf("RedisPubSub: ошибка публикации в канал %s (попытка %d/%d): %v",
			channel, attempt+1, p.maxRetries+1, err)

		// Ждем перед следующей попыткой
		if attempt < p.maxRetries {
			time.Sleep(p.retryInterval)
			// Увеличиваем интервал для следующей попытки
			p.retryInterval = time.Duration(float64(p.retryInterval) * 1.5)
		}
	}

	return fmt.Errorf("не удалось опубликовать сообщение после %d попыток: %w", p.maxRetries+1, err)
}

// Subscribe подписывается на указанный канал
func (p *RedisPubSub) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	log.Printf("RedisPubSub: подписка на канал %s", channel)

	// Создаем канал для сообщений с буфером
	msgCh := make(chan []byte, 64)

	// Получаем pubsub для указанного канала
	pubsub := p.client.Subscribe(p.ctx, channel)
	if _, err := pubsub.Receive(p.ctx); err != nil {
		close(msgCh)
		return nil, fmt.Errorf("ошибка подписки на канал %s: %w", channel, err)
	}

	// Сохраняем подписку в карте для возможности закрытия
	key := fmt.Sprintf("%s:%p", channel, msgCh)
	p.subscriptions.Store(key, pubsub)

	// Запускаем обработчик сообщений
	go func() {
		defer func() {
			// Закрываем PubSub и канал сообщений
			pubsub.Close()
			close(msgCh)
			// Удаляем подписку из карты
			p.subscriptions.Delete(key)
			log.Printf("RedisPubSub: подписка на канал %s закрыта", channel)
		}()

		redisChannel := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.ctx.Done():
				return
			case msg, ok := <-redisChannel:
				if !ok {
					// Канал закрыт
					return
				}

				// Отправляем сообщение
				select {
				case msgCh <- []byte(msg.Payload):
					// Успешно отправлено
				default:
					// Канал заполнен, пропускаем сообщение
					log.Printf("RedisPubSub: канал сообщений переполнен для %s, пропускаем сообщение", channel)
				}
			}
		}
	}()

	return msgCh, nil
}

// Close закрывает все подписки и освобождает ресурсы
func (p *RedisPubSub) Close() error {
	p.cancel()

	// Закрываем все подписки
	p.subscriptions.Range(func(key, value interface{}) bool {
		if pubsub, ok := value.(*redis.PubSub); ok {
			pubsub.Close()
		}
		return true
	})

	// Закрываем клиент Redis
	if p.client != nil {
		err := p.client.Close()
		if err != nil {
			log.Printf("RedisPubSub: ошибка при закрытии клиента Redis: %v", err)
			return err
		}
	}

	log.Printf("RedisPubSub: все ресурсы освобождены")
	return nil
}

// isRedisConnError проверяет, связана ли ошибка с проблемами подключения
func isRedisConnError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем строку ошибки на наличие слов, связанных с подключением
	errStr := err.Error()
	connectionErrors := []string{
		"connection", "timeout", "refused", "reset", "closed",
		"network", "dial", "connect", "i/o timeout",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errStr), connErr) {
			return true
		}
	}

	return false
}
