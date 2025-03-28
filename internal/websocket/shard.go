package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Shard представляет подмножество клиентов Hub
// Каждый шард обрабатывает свою группу клиентов независимо,
// что значительно улучшает производительность при большом числе соединений
type Shard struct {
	id         int           // Уникальный ID шарда
	clients    sync.Map      // Использование sync.Map вместо map[*Client]bool для лучшей производительности
	userMap    sync.Map      // Карта UserID -> Client
	broadcast  chan []byte   // Канал для широковещательных сообщений шарда
	register   chan *Client  // Канал для регистрации клиентов в шарде
	unregister chan *Client  // Канал для отмены регистрации клиентов из шарда
	done       chan struct{} // Сигнал для завершения работы шарда
	metrics    *ShardMetrics // Метрики производительности шарда
	parent     interface{}   // Ссылка на родительский хаб
	maxClients int           // Максимальное рекомендуемое количество клиентов в шарде

	// Добавляем поле для массовых отключений при высокой нагрузке
	massDisconnectQueue chan *Client // Очередь для массовых отключений
	// Канал для сигнализации о переполнении буфера отключений
	disconnectBufferAlert chan struct{}
	// Счетчик отложенных отключений
	pendingDisconnects int32
}

// ShardMetrics содержит метрики для отдельного шарда
type ShardMetrics struct {
	id                     int
	activeConnections      int64
	messagesSent           int64
	messagesReceived       int64
	connectionErrors       int64
	inactiveClientsRemoved int64
	lastCleanupTime        time.Time
	mu                     sync.RWMutex
}

// NewShard создает новый шард
func NewShard(id int, parent interface{}, maxClients int) *Shard {
	if maxClients <= 0 {
		maxClients = 2000 // Значение по умолчанию
	}

	// Создаем буфер для массовых отключений - увеличиваем до 1000
	// для обработки массовых отключений (до 10,000+ клиентов)
	massDisconnectQueueSize := 1000

	shard := &Shard{
		id:         id,
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		done:       make(chan struct{}),
		metrics: &ShardMetrics{
			id:              id,
			lastCleanupTime: time.Now(),
		},
		parent:                parent,
		maxClients:            maxClients,
		massDisconnectQueue:   make(chan *Client, massDisconnectQueueSize),
		disconnectBufferAlert: make(chan struct{}, 1),
		pendingDisconnects:    0,
	}

	log.Printf("[Шард %d] Создан с максимальным количеством клиентов %d", id, maxClients)
	return shard
}

// Run запускает цикл обработки сообщений шарда
func (s *Shard) Run() {
	// Запускаем обработчик массовых отключений
	go s.handleMassDisconnects()

	for {
		select {
		case client := <-s.register:
			s.handleRegister(client)
		case client := <-s.unregister:
			// При высокой нагрузке перенаправляем в очередь массовых отключений
			if atomic.LoadInt32(&s.pendingDisconnects) > 100 {
				// Пытаемся отправить в очередь без блокировки
				select {
				case s.massDisconnectQueue <- client:
					atomic.AddInt32(&s.pendingDisconnects, 1)
				default:
					// Если очередь заполнена, обрабатываем немедленно
					s.handleUnregister(client)
					// Сигнализируем о переполнении буфера
					select {
					case s.disconnectBufferAlert <- struct{}{}:
						log.Printf("[Шард %d] ВНИМАНИЕ: Буфер отключений переполнен!", s.id)
					default:
						// Уже есть сигнал, ничего не делаем
					}
				}
			} else {
				s.handleUnregister(client)
			}
		case message := <-s.broadcast:
			s.handleBroadcast(message)
		case <-s.done:
			log.Printf("[Шард %d] Получен сигнал завершения работы, останавливаемся", s.id)
			s.cleanupAllClients()
			return
		}
	}
}

// handleRegister регистрирует клиента в шарде
func (s *Shard) handleRegister(client *Client) {
	// Проверяем существующего клиента с тем же UserID
	if existingClient, loaded := s.userMap.LoadOrStore(client.UserID, client); loaded {
		oldClient, ok := existingClient.(*Client)
		if ok && oldClient != client {
			log.Printf("Shard %d: replacing client %s with new connection", s.id, client.UserID)

			// Создаем отложенное закрытие старого соединения
			go func() {
				time.Sleep(500 * time.Millisecond)
				s.clients.Delete(oldClient)
				s.userMap.CompareAndDelete(client.UserID, oldClient)

				if oldClient.conn != nil {
					oldClient.conn.Close()
				}
				close(oldClient.send)

				s.metrics.mu.Lock()
				s.metrics.activeConnections--
				s.metrics.mu.Unlock()
			}()
		}
	}

	// Регистрируем нового клиента
	s.clients.Store(client, true)
	client.lastActivity = time.Now()

	// Обновляем метрики
	s.metrics.mu.Lock()
	s.metrics.activeConnections++
	s.metrics.mu.Unlock()

	log.Printf("Shard %d: client %s registered", s.id, client.UserID)

	// Сигнал о завершении регистрации
	if client.registrationComplete != nil {
		select {
		case client.registrationComplete <- struct{}{}:
		default:
		}
	}
}

// handleUnregister удаляет клиента из шарда
func (s *Shard) handleUnregister(client *Client) {
	if _, ok := s.clients.LoadAndDelete(client); ok {
		// Удаляем из userMap, только если это тот же экземпляр
		if existingClient, loaded := s.userMap.Load(client.UserID); loaded {
			if existingClient == client {
				s.userMap.Delete(client.UserID)
			}
		}

		// Закрываем соединение
		if client.conn != nil {
			client.conn.Close()
		}
		close(client.send)

		// Обновляем метрики
		s.metrics.mu.Lock()
		s.metrics.activeConnections--
		s.metrics.mu.Unlock()

		log.Printf("Shard %d: client %s unregistered", s.id, client.UserID)
	}
}

// handleBroadcast отправляет сообщение всем клиентам в шарде
func (s *Shard) handleBroadcast(message []byte) {
	var clientCount int

	// Проверяем, есть ли в сообщении тип для фильтрации по подпискам
	var messageType string
	if len(message) > 2 { // Минимальная длина для JSON с полем type
		// Пытаемся распарсить JSON, чтобы получить тип сообщения
		var event struct {
			Type string `json:"type"`
		}
		// Используем UnmarshalJSON, который не модифицирует исходное сообщение
		if err := json.Unmarshal(message, &event); err == nil {
			messageType = event.Type
		}
	}

	// Флаг для проверки, является ли сообщение системным (отправляется всем)
	isSystemMessage := messageType == "system" || messageType == TOKEN_EXPIRED

	// Рассылаем сообщение всем клиентам
	s.clients.Range(func(key, value interface{}) bool {
		client, ok := key.(*Client)
		if !ok {
			return true // Пропускаем некорректные записи
		}

		// Если у сообщения есть тип и это не системное сообщение,
		// проверяем, подписан ли клиент на данный тип
		if messageType != "" && !isSystemMessage && !client.IsSubscribed(messageType) {
			return true // Клиент не подписан, пропускаем
		}

		clientCount++
		select {
		case client.send <- message:
			// Сообщение успешно отправлено в буфер клиента
		default:
			// Буфер клиента переполнен, отключаем клиента
			log.Printf("Shard %d: client %s buffer full, unregistering", s.id, client.UserID)
			s.clients.Delete(client)

			if existingClient, loaded := s.userMap.Load(client.UserID); loaded && existingClient == client {
				s.userMap.Delete(client.UserID)
			}

			if client.conn != nil {
				client.conn.Close()
			}
			close(client.send)

			// Обновляем метрики
			s.metrics.mu.Lock()
			s.metrics.activeConnections--
			s.metrics.connectionErrors++
			s.metrics.mu.Unlock()
		}
		return true
	})

	// Обновляем метрики
	if clientCount > 0 {
		s.metrics.mu.Lock()
		s.metrics.messagesSent += int64(clientCount)
		s.metrics.mu.Unlock()
	}

	// Добавляем тип сообщения в лог для более информативной отладки
	if messageType != "" {
		log.Printf("Shard %d: message of type %s broadcast to %d clients", s.id, messageType, clientCount)
	} else {
		log.Printf("Shard %d: message broadcast to %d clients", s.id, clientCount)
	}
}

// cleanupInactiveClients удаляет неактивных клиентов
func (s *Shard) cleanupInactiveClients() {
	now := time.Now()

	// Используем буферизированный канал для пакетной обработки отключений
	// Это позволяет избежать блокировок при массовых отключениях
	const batchSize = 100
	disconnectBatch := make(chan *Client, batchSize)

	s.metrics.mu.Lock()
	s.metrics.lastCleanupTime = now
	s.metrics.mu.Unlock()

	// Запускаем горутину для обработки отключений пакетами
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		batch := make([]*Client, 0, batchSize)
		inactiveCount := 0

		// Функция для обработки пакета отключений
		processBatch := func() {
			if len(batch) == 0 {
				return
			}

			// Обрабатываем всех клиентов в пакете
			for _, client := range batch {
				if client == nil {
					continue
				}

				s.clients.Delete(client)

				if existingClient, loaded := s.userMap.Load(client.UserID); loaded && existingClient == client {
					s.userMap.Delete(client.UserID)
				}

				if client.conn != nil {
					client.conn.Close()
				}
				close(client.send)
			}

			// Обновляем метрики один раз для всего пакета
			s.metrics.mu.Lock()
			s.metrics.inactiveClientsRemoved += int64(len(batch))
			s.metrics.activeConnections -= int64(len(batch))
			s.metrics.mu.Unlock()

			inactiveCount += len(batch)
			batch = batch[:0] // Очищаем слайс для переиспользования
		}

		// Обрабатываем клиентов из канала
		for client := range disconnectBatch {
			batch = append(batch, client)

			// Если пакет заполнен, обрабатываем его
			if len(batch) >= batchSize {
				processBatch()
			}
		}

		// Обрабатываем оставшихся клиентов
		processBatch()

		if inactiveCount > 0 {
			log.Printf("Shard %d: removed %d inactive clients", s.id, inactiveCount)
		}
	}()

	// Находим неактивных клиентов и отправляем их в канал
	s.clients.Range(func(key, value interface{}) bool {
		client, ok := key.(*Client)
		if !ok {
			return true
		}

		// Если клиент неактивен более 30 секунд, добавляем его в очередь на отключение
		if client.lastActivity.Add(30 * time.Second).Before(now) {
			select {
			case disconnectBatch <- client:
				// Успешно добавили клиента в очередь
			default:
				// Если канал полный, обрабатываем клиента напрямую
				s.clients.Delete(client)

				if existingClient, loaded := s.userMap.Load(client.UserID); loaded && existingClient == client {
					s.userMap.Delete(client.UserID)
				}

				if client.conn != nil {
					client.conn.Close()
				}
				close(client.send)

				// Обновляем метрики
				s.metrics.mu.Lock()
				s.metrics.inactiveClientsRemoved++
				s.metrics.activeConnections--
				s.metrics.mu.Unlock()
			}
		}
		return true
	})

	// Закрываем канал после завершения итерации
	close(disconnectBatch)

	// Ожидаем завершения обработки всех пакетов
	wg.Wait()
}

// cleanupAllClients закрывает все соединения перед остановкой шарда
func (s *Shard) cleanupAllClients() {
	s.clients.Range(func(key, value interface{}) bool {
		client, ok := key.(*Client)
		if !ok {
			return true
		}

		if client.conn != nil {
			client.conn.Close()
		}
		close(client.send)

		s.clients.Delete(client)
		return true
	})

	log.Printf("Shard %d: all clients cleanup completed", s.id)
}

// SendToUser отправляет сообщение конкретному пользователю в шарде
func (s *Shard) SendToUser(userID string, message []byte) bool {
	clientInterface, exists := s.userMap.Load(userID)
	if !exists {
		return false
	}

	client, ok := clientInterface.(*Client)
	if !ok {
		return false
	}

	select {
	case client.send <- message:
		// Обновляем метрики
		s.metrics.mu.Lock()
		s.metrics.messagesSent++
		s.metrics.mu.Unlock()
		return true
	default:
		// Буфер клиента переполнен, отключаем клиента
		log.Printf("Shard %d: client %s buffer full on direct message, unregistering", s.id, userID)
		s.clients.Delete(client)

		if existingClient, loaded := s.userMap.Load(client.UserID); loaded && existingClient == client {
			s.userMap.Delete(client.UserID)
		}

		if client.conn != nil {
			client.conn.Close()
		}
		close(client.send)

		// Обновляем метрики
		s.metrics.mu.Lock()
		s.metrics.activeConnections--
		s.metrics.connectionErrors++
		s.metrics.mu.Unlock()
		return false
	}
}

// BroadcastBytes рассылает байтовое сообщение всем клиентам в шарде
func (s *Shard) BroadcastBytes(message []byte) {
	select {
	case s.broadcast <- message:
		// Сообщение успешно отправлено в канал рассылки
	default:
		log.Printf("Shard %d: broadcast channel full, message dropped", s.id)
	}
}

// BroadcastJSON рассылает JSON-сообщение всем клиентам в шарде
func (s *Shard) BroadcastJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	s.BroadcastBytes(data)
	return nil
}

// GetMetrics возвращает метрики шарда
func (s *Shard) GetMetrics() map[string]interface{} {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	clientCount := s.GetClientCount()
	loadPercentage := float64(clientCount) / float64(s.maxClients) * 100

	disconnectMetrics := s.GetDisconnectionMetrics()

	return map[string]interface{}{
		"shard_id":            s.id,
		"active_connections":  clientCount,
		"max_clients":         s.maxClients,
		"messages_sent":       s.metrics.messagesSent,
		"messages_received":   s.metrics.messagesReceived,
		"connection_errors":   s.metrics.connectionErrors,
		"load_percentage":     loadPercentage,
		"last_cleanup":        s.metrics.lastCleanupTime.Format(time.RFC3339),
		"inactive_removed":    s.metrics.inactiveClientsRemoved,
		"disconnection_stats": disconnectMetrics,
	}
}

// GetClientCount возвращает количество активных клиентов в шарде
func (s *Shard) GetClientCount() int {
	var count int
	s.clients.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Close закрывает шард и освобождает ресурсы
func (s *Shard) Close() {
	close(s.done)
}

// Добавляем обработчик массовых отключений
func (s *Shard) handleMassDisconnects() {
	log.Printf("[Шард %d] Запущен обработчик массовых отключений", s.id)

	const minBatchSize = 50
	const maxBatchSize = 200
	currentBatchSize := minBatchSize
	batch := make([]*Client, 0, maxBatchSize)

	processBatch := func() {
		if len(batch) == 0 {
			return
		}

		start := time.Now()
		for _, client := range batch {
			s.handleUnregister(client)
			atomic.AddInt32(&s.pendingDisconnects, -1)
		}

		// Адаптивно меняем размер пакета в зависимости от времени выполнения
		elapsed := time.Since(start)
		if elapsed < 20*time.Millisecond && currentBatchSize < maxBatchSize {
			// Если обработка быстрая - увеличиваем размер пакета
			currentBatchSize = minBatch(currentBatchSize+20, maxBatchSize)
			log.Printf("[Шард %d] Увеличен размер пакета отключений до %d (время обработки: %v)",
				s.id, currentBatchSize, elapsed)
		} else if elapsed > 100*time.Millisecond && currentBatchSize > minBatchSize {
			// Если обработка медленная - уменьшаем размер пакета
			currentBatchSize = maxBatch(currentBatchSize-20, minBatchSize)
			log.Printf("[Шард %d] Уменьшен размер пакета отключений до %d (время обработки: %v)",
				s.id, currentBatchSize, elapsed)
		}

		// Очищаем обработанную партию
		batch = batch[:0]
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case client := <-s.massDisconnectQueue:
			batch = append(batch, client)
			if len(batch) >= currentBatchSize {
				processBatch()
			}
		case <-ticker.C:
			processBatch()
		case <-s.done:
			// Обрабатываем оставшиеся отключения перед завершением
			for {
				select {
				case client := <-s.massDisconnectQueue:
					s.handleUnregister(client)
				default:
					return
				}
			}
		}
	}
}

// Вспомогательные функции для работы с адаптивным размером пакета
func minBatch(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxBatch(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Добавляем метод для получения метрик отключений
func (s *Shard) GetDisconnectionMetrics() map[string]interface{} {
	return map[string]interface{}{
		"pending_disconnects":        atomic.LoadInt32(&s.pendingDisconnects),
		"disconnect_buffer_capacity": cap(s.massDisconnectQueue),
		"disconnect_buffer_used":     len(s.massDisconnectQueue),
		"buffer_alert_triggered":     len(s.disconnectBufferAlert) > 0,
	}
}
