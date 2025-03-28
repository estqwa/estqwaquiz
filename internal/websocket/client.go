package websocket

import (
	"bytes"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Время, которое разрешено писать сообщение клиенту.
	writeWait = 10 * time.Second

	// Время, которое разрешено клиенту читать следующее сообщение.
	// Уменьшено с 90 до 30 секунд для быстрого обнаружения отключений
	pongWait = 30 * time.Second

	// Периодичность отправки ping-сообщений клиенту.
	pingPeriod = (pongWait * 9) / 10

	// Максимальный размер сообщения
	maxMessageSize = 512

	// Размер буфера по умолчанию для каналов отправки сообщений клиенту
	defaultClientBufferSize = 64
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// ClientConfig содержит настройки для клиента
type ClientConfig struct {
	// BufferSize определяет размер буфера канала отправки сообщений
	BufferSize int

	// PingInterval определяет интервал между ping-сообщениями
	PingInterval time.Duration

	// PongWait определяет время ожидания pong-ответа
	PongWait time.Duration

	// WriteWait определяет тайм-аут для записи сообщений
	WriteWait time.Duration

	// MaxMessageSize определяет максимальный размер сообщения
	MaxMessageSize int64
}

// DefaultClientConfig возвращает конфигурацию клиента по умолчанию
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BufferSize:     defaultClientBufferSize,
		PingInterval:   pingPeriod,
		PongWait:       pongWait,
		WriteWait:      writeWait,
		MaxMessageSize: maxMessageSize,
	}
}

// Client является посредником между WebSocket соединением и hub.
type Client struct {
	// ID пользователя
	UserID string

	// Уникальный ID для каждого соединения
	ConnectionID string

	// Hub, к которому подключен клиент (может быть nil после миграции)
	hub interface{} // Изменено с *Hub на interface{} для поддержки ShardedHub

	// WebSocket соединение
	conn *websocket.Conn

	// Буферизованный канал для исходящих сообщений
	// Уменьшен размер буфера с 256 до 64 для экономии памяти
	send chan []byte

	// Время последней активности клиента
	lastActivity time.Time

	// Канал для ожидания завершения регистрации
	registrationComplete chan struct{}

	// Карта подписок на типы сообщений
	subscriptions sync.Map

	// Мьютекс для синхронизации доступа к подпискам
	subMutex sync.RWMutex

	// Роли клиента (например, "admin", "player", "spectator")
	roles map[string]bool
}

// NewClient создает нового клиента
func NewClient(hub interface{}, conn *websocket.Conn, userID string) *Client {
	connectionID := uuid.New().String()
	return &Client{
		hub:                  hub,
		conn:                 conn,
		send:                 make(chan []byte, 64), // Уменьшено с 256 до 64
		UserID:               userID,
		ConnectionID:         connectionID,
		lastActivity:         time.Now(),
		registrationComplete: make(chan struct{}, 1),
		roles:                make(map[string]bool),
	}
}

// NewClientWithConfig создает нового клиента с указанной конфигурацией
func NewClientWithConfig(hub interface{}, conn *websocket.Conn, userID string, config ClientConfig) *Client {
	connectionID := uuid.New().String()

	// Проверяем и исправляем недопустимые значения
	if config.BufferSize <= 0 {
		config.BufferSize = defaultClientBufferSize
	}

	return &Client{
		hub:                  hub,
		conn:                 conn,
		send:                 make(chan []byte, config.BufferSize),
		UserID:               userID,
		ConnectionID:         connectionID,
		lastActivity:         time.Now(),
		registrationComplete: make(chan struct{}, 1),
		roles:                make(map[string]bool),
	}
}

// readPump перекачивает сообщения от WebSocket соединения в hub.
func (c *Client) readPump(messageHandler func(message []byte, client *Client)) {
	defer func() {
		log.Printf("WebSocket readPump: client %s disconnected, unregistering from hub", c.UserID)

		// Обработка отключения в зависимости от типа хаба
		if h, ok := c.hub.(*Hub); ok {
			h.UnregisterClient(c)
		} else if sh, ok := c.hub.(*ShardedHub); ok {
			sh.UnregisterClient(c)
		}

		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		log.Printf("WebSocket: received pong from client %s", c.UserID)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastActivity = time.Now()
		return nil
	})

	log.Printf("WebSocket readPump: started for client %s", c.UserID)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.UserID, err)
			} else {
				log.Printf("WebSocket readPump: expected close error for client %s: %v", c.UserID, err)
			}
			break
		}
		c.lastActivity = time.Now()

		// Обновляем метрики полученных сообщений в зависимости от типа хаба
		if h, ok := c.hub.(*Hub); ok && h != nil {
			h.metrics.mu.Lock()
			h.metrics.messagesReceived++
			h.metrics.mu.Unlock()
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		log.Printf("WebSocket message received from client %s: %s", c.UserID, string(message))

		// Обработка полученного сообщения
		if messageHandler != nil {
			messageHandler(message, c)
		}
	}
}

// writePump перекачивает сообщения из hub клиенту через WebSocket соединение.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Printf("WebSocket writePump: client %s disconnected", c.UserID)
		ticker.Stop()
		c.conn.Close()
	}()

	log.Printf("WebSocket writePump: started for client %s", c.UserID)

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub закрыл канал.
				log.Printf("WebSocket writePump: hub closed channel for client %s", c.UserID)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("WebSocket writePump: error getting writer for client %s: %v", c.UserID, err)
				return
			}

			// Успешная отправка сообщения - обновляем время активности
			c.lastActivity = time.Now()

			log.Printf("WebSocket writePump: writing message to client %s: %s", c.UserID, string(message))
			w.Write(message)

			// Добавляем все ожидающие сообщения в текущий WebSocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				log.Printf("WebSocket writePump: error closing writer for client %s: %v", c.UserID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket writePump: error sending ping to client %s: %v", c.UserID, err)
				return
			}

			// Успешная отправка ping - обновляем время активности
			c.lastActivity = time.Now()

			log.Printf("WebSocket writePump: ping sent to client %s", c.UserID)
		}
	}
}

// StartPumps запускает goroutine для чтения и записи
func (c *Client) StartPumps(messageHandler func(message []byte, client *Client)) {
	if c.UserID == "" {
		log.Printf("WebSocket: client has no UserID, skipping registration")
		c.conn.Close()
		return
	}

	// Регистрируем клиента в хабе в зависимости от его типа
	if h, ok := c.hub.(*Hub); ok {
		log.Printf("WebSocket: registering client %s in legacy Hub", c.UserID)
		h.RegisterSync(c, c.registrationComplete)
	} else if sh, ok := c.hub.(*ShardedHub); ok {
		log.Printf("WebSocket: registering client %s in ShardedHub", c.UserID)
		sh.RegisterSync(c, c.registrationComplete)
	} else {
		log.Printf("WebSocket: unknown hub type for client %s, skipping registration", c.UserID)
		c.conn.Close()
		return
	}

	// Ожидаем завершения регистрации
	select {
	case <-c.registrationComplete:
		log.Printf("WebSocket: client %s fully registered, starting pumps", c.UserID)
	case <-time.After(5 * time.Second):
		log.Printf("WebSocket: timeout waiting for client %s registration", c.UserID)
		c.conn.Close()
		return
	}

	// Проверяем, что клиент все еще зарегистрирован
	clientExists := false

	if h, ok := c.hub.(*Hub); ok && h != nil {
		h.mu.RLock()
		clientExists = h.clients[c]
		h.mu.RUnlock()
	} else if sh, ok := c.hub.(*ShardedHub); ok && sh != nil {
		// Для ShardedHub проверка не требуется, так как шард сам управляет клиентами
		clientExists = true
	}

	if !clientExists {
		log.Printf("WebSocket: client %s was replaced before pumps started, skipping pumps", c.UserID)
		return
	}

	go c.writePump()
	go c.readPump(messageHandler)
}

// IsSubscribed проверяет, подписан ли клиент на указанный тип сообщений
func (c *Client) IsSubscribed(messageType string) bool {
	c.subMutex.RLock()
	defer c.subMutex.RUnlock()

	if messageType == "" {
		return true // Пустой тип означает все сообщения
	}

	// Проверяем, есть ли у клиента подписка на этот тип сообщений
	_, ok := c.subscriptions.Load(messageType)
	return ok
}

// Subscribe подписывает клиента на указанный тип сообщений
func (c *Client) Subscribe(messageType string) {
	if messageType == "" {
		return // Игнорируем пустые типы
	}

	c.subMutex.Lock()
	defer c.subMutex.Unlock()

	c.subscriptions.Store(messageType, true)
	log.Printf("WebSocket: клиент %s подписался на сообщения типа %s", c.UserID, messageType)
}

// Unsubscribe отменяет подписку клиента на указанный тип сообщений
func (c *Client) Unsubscribe(messageType string) {
	if messageType == "" {
		return // Игнорируем пустые типы
	}

	c.subMutex.Lock()
	defer c.subMutex.Unlock()

	c.subscriptions.Delete(messageType)
	log.Printf("WebSocket: клиент %s отписался от сообщений типа %s", c.UserID, messageType)
}

// GetSubscriptions возвращает список типов сообщений, на которые подписан клиент
func (c *Client) GetSubscriptions() []string {
	c.subMutex.RLock()
	defer c.subMutex.RUnlock()

	var subscriptions []string

	c.subscriptions.Range(func(key, value interface{}) bool {
		if messageType, ok := key.(string); ok {
			subscriptions = append(subscriptions, messageType)
		}
		return true
	})

	return subscriptions
}

// SubscribeToQuiz подписывает клиента на все типы сообщений викторины
func (c *Client) SubscribeToQuiz() {
	c.Subscribe(QUIZ_START)
	c.Subscribe(QUIZ_END)
	c.Subscribe(QUESTION_START)
	c.Subscribe(QUESTION_END)
	c.Subscribe(RESULT_UPDATE)
	log.Printf("WebSocket: клиент %s подписался на все сообщения викторины", c.UserID)
}

// HasRole проверяет, есть ли у клиента указанная роль
func (c *Client) HasRole(role string) bool {
	c.subMutex.RLock()
	defer c.subMutex.RUnlock()
	return c.roles[role]
}

// AddRole добавляет клиенту указанную роль
func (c *Client) AddRole(role string) {
	c.subMutex.Lock()
	defer c.subMutex.Unlock()
	c.roles[role] = true
	log.Printf("WebSocket: клиенту %s добавлена роль %s", c.UserID, role)
}

// RemoveRole удаляет у клиента указанную роль
func (c *Client) RemoveRole(role string) {
	c.subMutex.Lock()
	defer c.subMutex.Unlock()
	delete(c.roles, role)
	log.Printf("WebSocket: у клиента %s удалена роль %s", c.UserID, role)
}
