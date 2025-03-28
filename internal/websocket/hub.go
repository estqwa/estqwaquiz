package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Hub поддерживает набор активных клиентов и транслирует сообщения
// Это старый Hub, который сохраняется для обратной совместимости
// Новым кодом следует использовать ShardedHub
type Hub struct {
	// Зарегистрированные клиенты
	clients map[*Client]bool

	// Канал для входящей регистрации клиентов
	register chan *Client

	// Канал для отмены регистрации клиентов
	unregister chan *Client

	// Канал для широковещательных сообщений
	broadcast chan []byte

	// Маппинг UserID -> Client для прямой отправки
	userMap map[string]*Client

	// Мьютекс для потокобезопасной работы с картами
	mu sync.RWMutex

	// Каналы сигнализации о завершении регистрации клиентов
	registrationComplete map[*Client]chan struct{}

	// Мьютекс для потокобезопасной работы с картой регистраций
	registrationMu sync.RWMutex

	// Канал для завершения работы фоновых горутин
	done chan struct{}

	// Метрики для мониторинга
	metrics struct {
		totalConnections       int64
		activeConnections      int64
		messagesSent           int64
		messagesReceived       int64
		connectionErrors       int64
		inactiveClientsRemoved int64
		startTime              time.Time
		lastCleanupTime        time.Time

		// Мьютекс для безопасного обновления метрик
		mu sync.RWMutex
	}
}

// Проверка компилятором, что Hub реализует интерфейс HubInterface
var _ HubInterface = (*Hub)(nil)

// NewHub создает новый экземпляр Hub
// Устаревший метод, новому коду следует использовать NewShardedHub
func NewHub() *Hub {
	log.Println("ВНИМАНИЕ: Используется устаревший Hub. Рекомендуется перейти на ShardedHub для поддержки 10,000+ клиентов.")

	hub := &Hub{
		broadcast:            make(chan []byte),
		register:             make(chan *Client),
		unregister:           make(chan *Client),
		clients:              make(map[*Client]bool),
		userMap:              make(map[string]*Client),
		registrationComplete: make(map[*Client]chan struct{}),
		done:                 make(chan struct{}),
	}

	// Инициализация метрик
	hub.metrics.startTime = time.Now()

	return hub
}

// NewShardedHubFromConfig создаёт шардированный хаб с указанной конфигурацией
// Рекомендуется использовать вместо NewHub для новых приложений
func NewShardedHubFromConfig(config ShardedHubConfig) *ShardedHub {
	return NewShardedHub(config)
}

// NewShardedHubWithDefaults создаёт шардированный хаб с настройками по умолчанию
// Рекомендуется использовать вместо NewHub для новых приложений
func NewShardedHubWithDefaults() *ShardedHub {
	return NewShardedHub(DefaultShardedHubConfig())
}

// Run запускает цикл обработки сообщений Hub
func (h *Hub) Run() {
	// Запускаем фоновую очистку неактивных соединений
	go h.RunCleanupRoutine()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()

			// Обновляем метрики
			h.metrics.mu.Lock()
			h.metrics.totalConnections++
			h.metrics.activeConnections++
			h.metrics.mu.Unlock()

			if oldClient, exists := h.userMap[client.UserID]; exists && oldClient != client {
				log.Printf("Hub: detected existing connection for client %s", client.UserID)
				log.Println("WebSocket: Проверка существующего соединения для UserID", client.UserID)

				// 🔍 Проверяем, жив ли старый клиент перед его удалением
				err := oldClient.conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					log.Printf("Hub: detected inactive client %s, closing...", client.UserID)
					delete(h.clients, oldClient)
					delete(h.userMap, client.UserID)
					if oldClient.conn != nil {
						oldClient.conn.Close()
					}
					close(oldClient.send)

					// Обновляем метрики
					h.metrics.mu.Lock()
					h.metrics.activeConnections--
					h.metrics.connectionErrors++
					h.metrics.mu.Unlock()
				} else {
					log.Printf("Hub: replacing existing active client %s with new connection", client.UserID)
					// Улучшаем процесс замены: сначала регистрируем новый клиент,
					// а только потом закрываем старое соединение с задержкой

					// 1. Регистрируем нового клиента
					h.clients[client] = true
					h.userMap[client.UserID] = client
					client.lastActivity = time.Now()

					// 2. Создаем отложенное закрытие старого соединения
					oldClientCopy := oldClient // создаем копию, чтобы избежать проблем с гонками данных
					go func(oldClient *Client) {
						// Задержка перед закрытием, чтобы новое соединение успело установиться
						time.Sleep(500 * time.Millisecond)

						// Проверяем, что соединение не было закрыто другим процессом
						h.mu.Lock()
						_, stillExists := h.clients[oldClient]
						if stillExists {
							delete(h.clients, oldClient)
							// Проверяем, что в карте userMap все еще этот клиент
							if h.userMap[oldClient.UserID] == oldClient {
								delete(h.userMap, oldClient.UserID)
							}

							// Обновляем метрики
							h.metrics.mu.Lock()
							h.metrics.activeConnections--
							h.metrics.mu.Unlock()

							// Закрываем соединение и канал
							if oldClient.conn != nil {
								oldClient.conn.Close()
							}
							close(oldClient.send)
							log.Printf("Hub: delayed close of old client %s completed", oldClient.UserID)
						}
						h.mu.Unlock()
					}(oldClientCopy)

					// Пропускаем стандартную регистрацию ниже, так как мы уже зарегистрировали клиента
					h.mu.Unlock()

					// Отправляем сигнал о завершении регистрации
					h.registrationMu.RLock()
					if signalChan, ok := h.registrationComplete[client]; ok {
						select {
						case signalChan <- struct{}{}:
							log.Printf("Hub: sent registration completion signal to client %s", client.UserID)
						default:
							log.Printf("Hub: failed to send registration completion signal to client %s (channel buffer full)", client.UserID)
						}
					}
					h.registrationMu.RUnlock()

					return
				}
			}

			// ✅ Регистрация нового клиента
			h.clients[client] = true
			h.userMap[client.UserID] = client
			client.lastActivity = time.Now()
			log.Printf("Hub: client %s registered, total clients: %d", client.UserID, len(h.clients))
			h.mu.Unlock()

			// Отправляем сигнал о завершении регистрации
			h.registrationMu.RLock()
			if signalChan, ok := h.registrationComplete[client]; ok {
				select {
				case signalChan <- struct{}{}:
					log.Printf("Hub: sent registration completion signal to client %s", client.UserID)
				default:
					log.Printf("Hub: failed to send registration completion signal to client %s (channel buffer full)", client.UserID)
				}
			}
			h.registrationMu.RUnlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.userMap, client.UserID)
				log.Printf("Hub: client %s unregistered, total clients: %d", client.UserID, len(h.clients))

				// Обновляем метрики
				h.metrics.mu.Lock()
				h.metrics.activeConnections--
				h.metrics.mu.Unlock()

				// ✅ Проверяем, инициировал ли клиент закрытие
				if client.conn != nil {
					client.conn.Close()
				}

				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			log.Printf("Hub: broadcasting message to %d clients: %s", len(h.clients), string(message))

			// Обновляем метрики
			h.metrics.mu.Lock()
			h.metrics.messagesSent += int64(len(h.clients))
			h.metrics.mu.Unlock()

			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("Hub: failed to send message to client %s, unregistering", client.UserID)

					h.metrics.mu.Lock()
					h.metrics.connectionErrors++
					h.metrics.mu.Unlock()

					close(client.send)
					delete(h.clients, client)
					if client.UserID != "" {
						delete(h.userMap, client.UserID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// RunCleanupRoutine запускает периодическую очистку неактивных соединений
func (h *Hub) RunCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Printf("Hub: started cleanup routine")

	for {
		select {
		case <-ticker.C:
			h.mu.Lock()

			// Обновляем метрики
			h.metrics.mu.Lock()
			h.metrics.lastCleanupTime = time.Now()
			h.metrics.mu.Unlock()

			log.Printf("Hub: running cleanup check, current clients: %d", len(h.clients))
			now := time.Now()
			inactiveCount := 0

			for client := range h.clients {
				// Защита от nil-клиентов
				if client == nil {
					continue
				}

				// Если клиент не активен более 10 минут, закрываем соединение
				if client.lastActivity.Add(10 * time.Minute).Before(now) {
					log.Printf("Hub: cleanup - closing inactive client %s (last active: %v)",
						client.UserID, client.lastActivity.Format(time.RFC3339))

					delete(h.clients, client)
					delete(h.userMap, client.UserID)

					if client.conn != nil {
						client.conn.Close()
					}

					close(client.send)
					inactiveCount++
				}
			}

			if inactiveCount > 0 {
				h.metrics.mu.Lock()
				h.metrics.inactiveClientsRemoved += int64(inactiveCount)
				h.metrics.activeConnections -= int64(inactiveCount)
				h.metrics.mu.Unlock()

				log.Printf("Hub: cleanup removed %d inactive clients, remaining: %d",
					inactiveCount, len(h.clients))
			}

			h.mu.Unlock()

		case <-h.done:
			log.Printf("Hub: cleanup routine stopped")
			return
		}
	}
}

// GetMetrics возвращает текущие метрики Hub
func (h *Hub) GetMetrics() map[string]interface{} {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	h.mu.RLock()
	currentActiveConnections := len(h.clients)
	h.mu.RUnlock()

	uptime := time.Since(h.metrics.startTime).Seconds()

	return map[string]interface{}{
		"total_connections":        h.metrics.totalConnections,
		"active_connections":       currentActiveConnections,
		"messages_sent":            h.metrics.messagesSent,
		"messages_received":        h.metrics.messagesReceived,
		"connection_errors":        h.metrics.connectionErrors,
		"inactive_clients_removed": h.metrics.inactiveClientsRemoved,
		"uptime_seconds":           uptime,
		"last_cleanup":             h.metrics.lastCleanupTime.Format(time.RFC3339),
	}
}

// Close закрывает все ресурсы и горутины Hub
func (h *Hub) Close() {
	close(h.done)

	h.mu.Lock()
	for client := range h.clients {
		if client.conn != nil {
			client.conn.Close()
		}
		close(client.send)
	}

	// Очищаем все карты
	h.clients = make(map[*Client]bool)
	h.userMap = make(map[string]*Client)
	h.mu.Unlock()

	h.registrationMu.Lock()
	h.registrationComplete = make(map[*Client]chan struct{})
	h.registrationMu.Unlock()

	log.Printf("Hub: closed")
}

// RegisterSync регистрирует клиента и ожидает завершения регистрации
func (h *Hub) RegisterSync(client *Client, done chan struct{}) {
	// Создаем канал для прямого уведомления от хаба
	syncChan := make(chan struct{}, 1)

	// Регистрируем этот канал в структуре Hub
	h.registrationMu.Lock()
	h.registrationComplete[client] = syncChan
	h.registrationMu.Unlock()

	// Отправляем в канал регистрации
	h.register <- client

	// Добавляем таймаут ожидания сигнала регистрации
	select {
	case <-syncChan:
		log.Printf("Hub: client %s successfully registered", client.UserID)
	case <-time.After(3 * time.Second): // Даем чуть больше времени
		log.Printf("Hub: timeout while registering client %s - possible race condition", client.UserID)
	}

	// Удаляем канал из карты после получения сигнала
	h.registrationMu.Lock()
	delete(h.registrationComplete, client)
	h.registrationMu.Unlock()

	// Передаем сигнал в StartPumps
	done <- struct{}{}
}

// RegisterClient регистрирует нового клиента
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient отменяет регистрацию клиента
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// Broadcast отправляет сообщение всем клиентам
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastJSON отправляет структуру JSON всем клиентам
func (h *Hub) BroadcastJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// Логируем сообщение и количество получателей
	h.mu.RLock()
	clientCount := len(h.clients)
	h.mu.RUnlock()

	log.Printf("Hub: broadcasting message to %d clients: %s", clientCount, string(data[:min(200, len(data))]))

	h.broadcast <- data
	return nil
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SendToUser отправляет сообщение конкретному пользователю
func (h *Hub) SendToUser(userID string, message []byte) bool {
	h.mu.RLock()
	client, exists := h.userMap[userID]
	h.mu.RUnlock()

	if exists {
		log.Printf("Hub: sending message to user %s: %s", userID, string(message))
		select {
		case client.send <- message:
			return true
		default:
			log.Printf("Hub: failed to send message to user %s, buffer full", userID)
			h.mu.Lock()
			delete(h.clients, client)
			delete(h.userMap, userID)
			close(client.send)
			h.mu.Unlock()
			return false
		}
	}
	log.Printf("Hub: user %s not found", userID)
	return false
}

// SendJSONToUser отправляет структуру JSON конкретному пользователю
func (h *Hub) SendJSONToUser(userID string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Hub: error marshaling JSON for user %s: %v", userID, err)
		return err
	}
	if !h.SendToUser(userID, data) {
		log.Printf("Hub: failed to send JSON to user %s", userID)
		return err
	}
	return nil
}

// ClientCount возвращает количество подключенных клиентов
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
