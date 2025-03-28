package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Event представляет структуру WebSocket-сообщения
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// HubInterface определяет общий интерфейс для Hub и ShardedHub
type HubInterface interface {
	// BroadcastJSON отправляет структуру JSON всем клиентам
	BroadcastJSON(v interface{}) error

	// SendJSONToUser отправляет структуру JSON конкретному пользователю
	SendJSONToUser(userID string, v interface{}) error

	// SendToUser отправляет байтовое сообщение конкретному пользователю
	SendToUser(userID string, message []byte) bool

	// GetMetrics возвращает метрики хаба
	GetMetrics() map[string]interface{}

	// ClientCount возвращает количество подключенных клиентов
	ClientCount() int
}

// Добавляем структуру для поддержки приоритетных очередей сообщений
type messagePriorityQueue struct {
	// Очереди по приоритетам
	queues map[int][]interface{}

	// Ёмкость буфера для каждого приоритета
	capacities map[int]int

	// Статистика отбрасываний по приоритетам
	dropped map[int]int64

	// Метрики рассылки
	metrics struct {
		enqueued int64
		dequeued int64
		dropped  int64
	}

	mu sync.RWMutex
}

// Создает новую приоритетную очередь сообщений
func newMessagePriorityQueue() *messagePriorityQueue {
	q := &messagePriorityQueue{
		queues:     make(map[int][]interface{}),
		capacities: make(map[int]int),
		dropped:    make(map[int]int64),
	}

	// Устанавливаем ёмкость по приоритетам
	q.capacities[PriorityLow] = 100       // Низкий - до 100 сообщений
	q.capacities[PriorityNormal] = 500    // Нормальный - до 500 сообщений
	q.capacities[PriorityHigh] = 1000     // Высокий - до 1000 сообщений
	q.capacities[PriorityCritical] = 5000 // Критический - до 5000 сообщений

	// Инициализируем очереди
	q.queues[PriorityLow] = make([]interface{}, 0, 20)
	q.queues[PriorityNormal] = make([]interface{}, 0, 50)
	q.queues[PriorityHigh] = make([]interface{}, 0, 100)
	q.queues[PriorityCritical] = make([]interface{}, 0, 200)

	return q
}

// Добавляет сообщение в очередь с учетом приоритета
func (q *messagePriorityQueue) enqueue(priority int, message interface{}) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Проверяем наличие очереди для приоритета
	if _, ok := q.queues[priority]; !ok {
		q.queues[priority] = make([]interface{}, 0, 50)
	}

	// Проверяем заполнение очереди
	capacity, ok := q.capacities[priority]
	if !ok {
		capacity = 100 // По умолчанию ограничиваем до 100
	}

	// Если очередь переполнена, отбрасываем сообщение
	if len(q.queues[priority]) >= capacity {
		if _, ok := q.dropped[priority]; !ok {
			q.dropped[priority] = 0
		}
		q.dropped[priority]++
		q.metrics.dropped++
		return false
	}

	// Добавляем сообщение в очередь
	q.queues[priority] = append(q.queues[priority], message)
	q.metrics.enqueued++
	return true
}

// Извлекает сообщение с наивысшим приоритетом
func (q *messagePriorityQueue) dequeue() (interface{}, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Проверяем очереди от высокого приоритета к низкому
	for priority := PriorityCritical; priority >= PriorityLow; priority-- {
		if queue, ok := q.queues[priority]; ok && len(queue) > 0 {
			message := queue[0]
			q.queues[priority] = queue[1:]
			q.metrics.dequeued++
			return message, true
		}
	}

	// Очередь пуста
	return nil, false
}

// Возвращает статистику очереди
func (q *messagePriorityQueue) stats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	queueSizes := make(map[string]int)
	droppedStats := make(map[string]int64)

	// Собираем размеры очередей по приоритетам
	for priority, queue := range q.queues {
		var priorityName string
		switch priority {
		case PriorityLow:
			priorityName = "low"
		case PriorityNormal:
			priorityName = "normal"
		case PriorityHigh:
			priorityName = "high"
		case PriorityCritical:
			priorityName = "critical"
		default:
			priorityName = fmt.Sprintf("unknown_%d", priority)
		}

		queueSizes[priorityName] = len(queue)
		if dropped, ok := q.dropped[priority]; ok {
			droppedStats[priorityName] = dropped
		}
	}

	return map[string]interface{}{
		"queue_sizes":   queueSizes,
		"dropped":       droppedStats,
		"total_queued":  q.metrics.enqueued - q.metrics.dequeued,
		"enqueued":      q.metrics.enqueued,
		"dequeued":      q.metrics.dequeued,
		"total_dropped": q.metrics.dropped,
	}
}

// Manager обрабатывает WebSocket сообщения
type Manager struct {
	hub            HubInterface
	messageHandler map[string]func(data json.RawMessage, client *Client) error

	// Очередь приоритетов
	priorityQueue *messagePriorityQueue

	// Канал для обработки очереди сообщений
	queuedMessages chan struct{}

	// WaitGroup для ожидания завершения обработки
	wg sync.WaitGroup

	// Счетчики доставки сообщений
	messageStats struct {
		sentCount     int64
		failedCount   int64
		droppedCount  int64
		priorityStats map[int]map[string]int64 // Статистика по приоритетам
		mu            sync.RWMutex
	}
}

// PrioritizedEvent представляет событие с приоритетом для правильной обработки
type PrioritizedEvent struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	Priority int         `json:"-"` // Не сериализуется в JSON
}

// NewManager создает новый менеджер WebSocket
func NewManager(hub HubInterface) *Manager {
	manager := &Manager{
		hub:            hub,
		messageHandler: make(map[string]func(data json.RawMessage, client *Client) error),
		priorityQueue:  newMessagePriorityQueue(),
		queuedMessages: make(chan struct{}, 1),
	}

	// Инициализируем статистику
	manager.messageStats.priorityStats = make(map[int]map[string]int64)
	for _, priority := range []int{PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical} {
		manager.messageStats.priorityStats[priority] = map[string]int64{
			"sent":    0,
			"failed":  0,
			"dropped": 0,
		}
	}

	// Запускаем обработчик очереди
	manager.wg.Add(1)
	go manager.processMessageQueue()

	return manager
}

// RegisterHandler регистрирует обработчик для определенного типа сообщений
func (m *Manager) RegisterHandler(eventType string, handler func(data json.RawMessage, client *Client) error) {
	m.messageHandler[eventType] = handler
	log.Printf("[WebSocketManager] Зарегистрирован обработчик для сообщений типа: %s", eventType)
}

// HandleMessage обрабатывает входящее сообщение
func (m *Manager) HandleMessage(message []byte, client *Client) {
	// Разбираем сообщение
	var event struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Error parsing message: %v", err)
		m.sendErrorToClient(client, "message_parse_error", fmt.Sprintf("Ошибка разбора сообщения: %v", err))
		return
	}

	// Ищем обработчик для данного типа сообщения
	handler, exists := m.messageHandler[event.Type]
	if !exists {
		log.Printf("No handler registered for event type: %s", event.Type)
		m.sendErrorToClient(client, "unsupported_message_type", fmt.Sprintf("Неподдерживаемый тип сообщения: %s", event.Type))
		return
	}

	// Вызываем обработчик
	if err := handler(event.Data, client); err != nil {
		log.Printf("Error handling message of type %s: %v", event.Type, err)

		// Отправляем сообщение об ошибке клиенту
		m.sendErrorToClient(client, "processing_error", fmt.Sprintf("Ошибка обработки %s: %v", event.Type, err))
	}
}

// sendErrorToClient отправляет сообщение об ошибке клиенту
func (m *Manager) sendErrorToClient(client *Client, code string, message string) {
	errorEvent := Event{
		Type: "error",
		Data: map[string]string{
			"code":    code,
			"message": message,
		},
	}

	if data, err := json.Marshal(errorEvent); err == nil {
		client.send <- data
	}
}

// BroadcastEvent отправляет событие всем клиентам
func (m *Manager) BroadcastEvent(eventType string, data interface{}) error {
	// Определяем приоритет сообщения
	priority, exists := MessagePriorityMap[eventType]
	if !exists {
		priority = PriorityNormal // По умолчанию - нормальный приоритет
	}

	event := Event{
		Type: eventType,
		Data: data,
	}

	// Увеличиваем счетчики отправленных сообщений
	m.updateMessageStats(eventType, priority, true)

	// Для высокоприоритетных сообщений используем прямую отправку
	if priority >= PriorityHigh {
		return m.broadcastHighPriorityJSON(event)
	}

	// Для низкоприоритетных сообщений используем очередь
	if m.priorityQueue.enqueue(priority, &prioritizedBroadcast{
		event:    event,
		priority: priority,
	}) {
		// Сигнализируем о новом сообщении в очереди
		select {
		case m.queuedMessages <- struct{}{}:
		default:
			// Канал уже имеет сигнал, не блокируемся
		}
		return nil
	}

	// Сообщение отброшено из-за переполнения очереди
	m.messageStats.mu.Lock()
	m.messageStats.droppedCount++
	if stats, ok := m.messageStats.priorityStats[priority]; ok {
		stats["dropped"]++
	}
	m.messageStats.mu.Unlock()

	return fmt.Errorf("сообщение отброшено из-за переполнения очереди (приоритет %d)", priority)
}

// updateMessageStats обновляет статистику отправки сообщений
func (m *Manager) updateMessageStats(eventType string, priority int, success bool) {
	m.messageStats.mu.Lock()
	defer m.messageStats.mu.Unlock()

	if success {
		m.messageStats.sentCount++
		m.messageStats.priorityStats[priority]["sent"]++
	} else {
		m.messageStats.failedCount++
		m.messageStats.priorityStats[priority]["failed"]++

		// Для критичных сообщений логируем ошибку более заметно
		if priority >= PriorityCritical {
			log.Printf("[КРИТИЧЕСКАЯ ОШИБКА] Не удалось отправить сообщение типа %s", eventType)
		}
	}
}

// broadcastHighPriorityJSON отправляет высокоприоритетные сообщения с повышенной надежностью
func (m *Manager) broadcastHighPriorityJSON(v interface{}) error {
	// Сериализуем сообщение в JSON
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// Если у нас ShardedHub, используем его специфические методы
	if sh, ok := m.hub.(*ShardedHub); ok {
		// Отправляем сообщение с подтверждением доставки ключевым шардам
		log.Printf("[WebSocket] Высокоприоритетная рассылка через ShardedHub")

		// Проверяем, поддерживает ли ShardedHub приоритизированную рассылку
		if method, ok := interface{}(sh).(interface{ BroadcastPrioritized([]byte) error }); ok {
			return method.BroadcastPrioritized(data)
		}

		// Если не поддерживает, используем стандартный метод
		if err := sh.BroadcastJSON(v); err != nil {
			// Создаем и отправляем алерт, если такая функция доступна
			if alertSender, ok := interface{}(sh).(interface {
				SendAlert(alertType, severity, message string, metadata map[string]interface{})
			}); ok {
				event, ok := v.(Event)
				eventType := "unknown"
				if ok {
					eventType = event.Type
				}

				alertSender.SendAlert("message_loss", "critical",
					"Не удалось доставить высокоприоритетное сообщение",
					map[string]interface{}{
						"event_type": eventType,
						"error":      err.Error(),
					})
			}
			return err
		}
		return nil
	}

	// Для обычного Hub просто используем стандартный метод
	log.Printf("[WebSocket] Высокоприоритетная рассылка через обычный Hub")
	return m.hub.BroadcastJSON(v)
}

// SendEventToUser отправляет событие конкретному пользователю
func (m *Manager) SendEventToUser(userID string, eventType string, data interface{}) error {
	// Определяем приоритет сообщения
	priority, exists := MessagePriorityMap[eventType]
	if !exists {
		priority = PriorityNormal // По умолчанию - нормальный приоритет
	}

	event := Event{
		Type: eventType,
		Data: data,
	}

	// Для высокоприоритетных и критичных сообщений используем прямую отправку
	if priority >= PriorityHigh {
		return m.sendHighPriorityToUser(userID, event)
	}

	// Для низкоприоритетных сообщений используем очередь
	if m.priorityQueue.enqueue(priority, &prioritizedDirectMessage{
		userID:   userID,
		event:    event,
		priority: priority,
	}) {
		// Сигнализируем о новом сообщении в очереди
		select {
		case m.queuedMessages <- struct{}{}:
		default:
			// Канал уже имеет сигнал, не блокируемся
		}
		return nil
	}

	// Сообщение отброшено из-за переполнения очереди
	m.messageStats.mu.Lock()
	m.messageStats.droppedCount++
	if stats, ok := m.messageStats.priorityStats[priority]; ok {
		stats["dropped"]++
	}
	m.messageStats.mu.Unlock()

	return fmt.Errorf("сообщение для пользователя %s отброшено из-за переполнения очереди (приоритет %d)",
		userID, priority)
}

// sendHighPriorityToUser отправляет высокоприоритетное сообщение пользователю с повторными попытками
func (m *Manager) sendHighPriorityToUser(userID string, v interface{}) error {
	// Сериализуем сообщение в JSON
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// Настройка повторных попыток
	maxRetries := 3
	retryDelay := 500 * time.Millisecond

	// Отправляем с повторными попытками
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if m.hub.SendToUser(userID, data) {
			// Успешная отправка
			if attempt > 1 {
				log.Printf("[WebSocket] Высокоприоритетное сообщение доставлено пользователю %s с %d попытки",
					userID, attempt)
			}
			return nil
		}

		// Если это не последняя попытка, ждем и пробуем снова
		if attempt < maxRetries {
			log.Printf("[WebSocket] Повторная попытка (%d/%d) отправки высокоприоритетного сообщения пользователю %s",
				attempt, maxRetries, userID)
			time.Sleep(retryDelay)

			// Увеличиваем задержку для следующей попытки
			retryDelay *= 2
		}
	}

	log.Printf("[WebSocket] Не удалось доставить высокоприоритетное сообщение пользователю %s после %d попыток",
		userID, maxRetries)

	return fmt.Errorf("не удалось доставить сообщение пользователю %s", userID)
}

// SendTokenExpirationWarning отправляет пользователю предупреждение о скором истечении срока действия токена
func (m *Manager) SendTokenExpirationWarning(userID string, expiresIn int) {
	// Создаем сообщение
	message := map[string]interface{}{
		"type": TOKEN_EXPIRE_SOON,
		"data": map[string]interface{}{
			"expires_in": expiresIn,
			"unit":       "seconds",
		},
	}

	// Отправляем пользователю
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocketManager] Ошибка при сериализации предупреждения о токене: %v", err)
		return
	}

	sent := m.hub.SendToUser(userID, jsonMessage)
	if sent {
		log.Printf("[WebSocketManager] Отправлено предупреждение о истечении токена пользователю ID=%s", userID)
	} else {
		log.Printf("[WebSocketManager] Не удалось отправить предупреждение о истечении токена пользователю ID=%s", userID)
	}
}

// SendTokenExpiredNotification отправляет пользователю уведомление о истечении срока действия токена
func (m *Manager) SendTokenExpiredNotification(userID string) {
	// Создаем сообщение
	message := map[string]interface{}{
		"type": TOKEN_EXPIRED,
		"data": map[string]interface{}{
			"message": "Срок действия токена истек. Необходимо выполнить повторный вход.",
		},
	}

	// Отправляем пользователю
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocketManager] Ошибка при сериализации уведомления о истечении токена: %v", err)
		return
	}

	sent := m.hub.SendToUser(userID, jsonMessage)
	if sent {
		log.Printf("[WebSocketManager] Отправлено уведомление о истечении токена пользователю ID=%s", userID)
	} else {
		log.Printf("[WebSocketManager] Не удалось отправить уведомление о истечении токена пользователю ID=%s", userID)
	}
}

// GetMetrics возвращает текущие метрики WebSocket-системы
func (m *Manager) GetMetrics() map[string]interface{} {
	m.messageStats.mu.RLock()
	defer m.messageStats.mu.RUnlock()

	// Собираем статистику по приоритетам
	priorityStats := make(map[string]map[string]int64)
	for priority, stats := range m.messageStats.priorityStats {
		var priorityName string
		switch priority {
		case PriorityLow:
			priorityName = "low"
		case PriorityNormal:
			priorityName = "normal"
		case PriorityHigh:
			priorityName = "high"
		case PriorityCritical:
			priorityName = "critical"
		default:
			priorityName = fmt.Sprintf("unknown_%d", priority)
		}
		priorityStats[priorityName] = stats
	}

	// Добавляем метрики очереди приоритетов
	queueStats := m.priorityQueue.stats()

	return map[string]interface{}{
		"messages_sent":        m.messageStats.sentCount,
		"messages_sent_failed": m.messageStats.failedCount,
		"messages_dropped":     m.messageStats.droppedCount,
		"priority_stats":       priorityStats,
		"queue_stats":          queueStats,
		"client_count":         m.hub.ClientCount(),
	}
}

// GetClientCount возвращает количество подключенных клиентов
func (m *Manager) GetClientCount() int {
	return m.hub.ClientCount()
}

// SubscribeClientToTypes подписывает клиента на указанные типы сообщений
func (m *Manager) SubscribeClientToTypes(client *Client, messageTypes []string) {
	for _, msgType := range messageTypes {
		client.Subscribe(msgType)
	}
}

// SubscribeClientToQuiz подписывает клиента на все сообщения викторины
func (m *Manager) SubscribeClientToQuiz(client *Client) {
	client.SubscribeToQuiz()
}

// UnsubscribeClientFromTypes отменяет подписку клиента на указанные типы сообщений
func (m *Manager) UnsubscribeClientFromTypes(client *Client, messageTypes []string) {
	for _, msgType := range messageTypes {
		client.Unsubscribe(msgType)
	}
}

// BroadcastEventToSubscribers отправляет событие только подписанным клиентам
func (m *Manager) BroadcastEventToSubscribers(eventType string, data interface{}) error {
	event := Event{
		Type: eventType,
		Data: data,
	}

	// Определяем приоритет сообщения
	priority, exists := MessagePriorityMap[eventType]
	if !exists {
		priority = PriorityNormal
	}

	log.Printf("[WebSocket] Отправка события <%s> (приоритет: %d) подписанным клиентам",
		eventType, priority)

	// Для высокоприоритетных сообщений используем специальный метод
	if priority >= PriorityHigh {
		// Здесь используем стандартный метод, так как фильтрация происходит на уровне шардов
		return m.broadcastHighPriorityJSON(event)
	}

	return m.hub.BroadcastJSON(event)
}

// BroadcastQuizStart рассылает сообщение о начале викторины
func (m *Manager) BroadcastQuizStart(quizID string, data interface{}) error {
	log.Printf("[WebSocket] Рассылка уведомления о начале викторины %s", quizID)
	return m.BroadcastEventToSubscribers(QUIZ_START, data)
}

// BroadcastQuizEnd рассылает сообщение о завершении викторины
func (m *Manager) BroadcastQuizEnd(quizID string, data interface{}) error {
	log.Printf("[WebSocket] Рассылка уведомления о завершении викторины %s", quizID)
	return m.BroadcastEventToSubscribers(QUIZ_END, data)
}

// BroadcastQuestionStart рассылает сообщение о начале вопроса
func (m *Manager) BroadcastQuestionStart(quizID string, questionNumber int, data interface{}) error {
	log.Printf("[WebSocket] Рассылка уведомления о начале вопроса %d в викторине %s",
		questionNumber, quizID)
	return m.BroadcastEventToSubscribers(QUESTION_START, data)
}

// BroadcastQuestionEnd рассылает сообщение о завершении вопроса
func (m *Manager) BroadcastQuestionEnd(quizID string, questionNumber int, data interface{}) error {
	log.Printf("[WebSocket] Рассылка уведомления о завершении вопроса %d в викторине %s",
		questionNumber, quizID)
	return m.BroadcastEventToSubscribers(QUESTION_END, data)
}

// BroadcastResults рассылает обновление результатов
func (m *Manager) BroadcastResults(quizID string, data interface{}) error {
	log.Printf("[WebSocket] Рассылка обновления результатов викторины %s", quizID)
	return m.BroadcastEventToSubscribers(RESULT_UPDATE, data)
}

// Обработка очереди сообщений
func (m *Manager) processMessageQueue() {
	defer m.wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.queuedMessages:
			// Обрабатываем сообщения из очереди
			m.drainMessageQueue()
		case <-ticker.C:
			// Периодически проверяем очередь
			m.drainMessageQueue()
		}
	}
}

// Обработка сообщений из очереди с учетом приоритетов
func (m *Manager) drainMessageQueue() {
	const batchSize = 20
	processed := 0

	// Обрабатываем до batchSize сообщений за один вызов
	for processed < batchSize {
		message, ok := m.priorityQueue.dequeue()
		if !ok {
			// Очередь пуста
			break
		}

		// Определяем тип сообщения и выполняем отправку
		switch msg := message.(type) {
		case *prioritizedBroadcast:
			// Широковещательное сообщение
			m.hub.BroadcastJSON(msg.event)
		case *prioritizedDirectMessage:
			// Личное сообщение
			m.hub.SendJSONToUser(msg.userID, msg.event)
		}

		processed++
	}
}

// Структуры для приоритетной очереди
type prioritizedBroadcast struct {
	event    interface{}
	priority int
}

type prioritizedDirectMessage struct {
	userID   string
	event    interface{}
	priority int
}
