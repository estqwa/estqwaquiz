# WebSocket Подсистема: Архитектура и особенности реализации

## Обзор

WebSocket подсистема Trivia API обеспечивает двунаправленную коммуникацию в реальном времени с высокой производительностью и масштабируемостью. Она разработана для одновременного обслуживания десятков тысяч соединений с низкой латентностью и высокой отказоустойчивостью.

## Архитектура

Подсистема построена на многоуровневой архитектуре с разделением на несколько ключевых компонентов:

```
                        ┌───────────────────────┐
                        │   WebSocket Handler   │
                        └───────────┬───────────┘
                                    │
                                    ▼
                        ┌───────────────────────┐
                        │   Manager (API/Core)  │
                        └───────────┬───────────┘
                                    │
                        ┌───────────┴───────────┐
                        │     ShardedHub        │
                        └─┬─────────────────────┘
           ┌──────────────┼───────────────┬────────────────┐
           │              │               │                │
     ┌─────▼─────┐  ┌─────▼─────┐   ┌─────▼─────┐    ┌─────▼─────┐
     │  Shard 0  │  │  Shard 1  │   │  Shard 2  │    │  Shard N  │
     └─────┬─────┘  └─────┬─────┘   └─────┬─────┘    └─────┬─────┘
           │              │               │                │
      ┌────┴───┐     ┌────┴───┐      ┌────┴───┐       ┌────┴───┐
      │ Clients │     │ Clients │      │ Clients │       │ Clients │
      └────────┘     └────────┘      └────────┘       └────────┘
           │              │               │                │
           └──────────────┼───────────────┼────────────────┘
                          │               │
                    ┌─────▼───────────────▼─────┐
                    │     Redis PubSub Hub      │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │      Other Instances      │
                    └───────────────────────────┘
```

### Основные компоненты

#### Client (client.go)

Представляет соединение с конкретным пользователем и обеспечивает:
- Чтение сообщений из WebSocket соединения
- Запись сообщений в WebSocket соединение
- Управление буферами входящих и исходящих сообщений
- Обработку тайм-аутов и закрытий соединений
- Хранение пользовательских метаданных (ID, подписки)
- Периодические пинги для проверки активности соединения

Структура клиента:
```go
type Client struct {
    ID           string
    UserID       uint
    Hub          Hub
    Conn         *websocket.Conn
    Send         chan []byte
    Closed       bool
    ClosedMutex  sync.RWMutex
    Subscriptions map[string]bool
    LastActivity time.Time
    DeviceInfo   string
}
```

#### Hub (hub.go)

Базовый компонент для управления клиентскими соединениями:
- Регистрация новых клиентов
- Отмена регистрации отключившихся клиентов
- Широковещательная рассылка сообщений
- Отправка сообщений конкретным пользователям
- Поддержка базовой статистики по соединениям

Структура простого Hub:
```go
type Hub struct {
    Clients      map[string]*Client
    Register     chan *Client
    Unregister   chan *Client
    Broadcast    chan []byte
    ClientsMutex sync.RWMutex
    MetricsMutex sync.RWMutex
}
```

#### ShardedHub (sharded_hub.go)

Расширение Hub с поддержкой шардирования:
- Распределение клиентов по шардам для повышения производительности
- Динамическое балансирование нагрузки между шардами
- Изоляция операций в отдельных шардах для предотвращения блокировок
- Поддержка миграции клиентов между шардами
- Сбор и агрегация метрик по каждому шарду

Структура ShardedHub:
```go
type ShardedHub struct {
    Shards       []*Hub
    ShardCount   int
    ClientCount  int64
    Distribution map[int]int
    Mutex        sync.RWMutex
    Options      ShardOptions
}

type ShardOptions struct {
    MaxClientsPerShard int
    ShardBalanceThreshold float64
    EnableAutoRebalance bool
    MetricsEnabled bool
}
```

#### Shard (shard.go)

Контейнер для группы клиентов, обеспечивающий:
- Изоляцию событий регистрации/отмены регистрации клиентов
- Параллельную обработку сообщений в разных шардах
- Локальную статистику и мониторинг
- Эффективное управление ресурсами
- Обработку пиковых нагрузок в отдельных шардах

#### Manager (manager.go)

Высокоуровневый компонент, обеспечивающий бизнес-логику WebSocket:
- Управление подключением/отключением пользователей
- Обработка и маршрутизация сообщений
- Регистрация обработчиков разных типов сообщений
- Интеграция с Redis для кластеризации
- Приоритизация сообщений
- Управление очередями сообщений
- Сбор метрик и телеметрии

```go
type Manager struct {
    hub           *ShardedHub
    pubsub        *PubSub
    handlers      map[string]MessageHandler
    metrics       *Metrics
    isClustered   bool
    eventChannels map[string]chan Event
    workerPool    *WorkerPool
}

type MessageHandler func(client *Client, data json.RawMessage) error
```

#### PubSub (pubsub.go)

Обеспечивает кластеризацию через Redis:
- Синхронизация сообщений между разными экземплярами сервера
- Репликация широковещательных сообщений между узлами
- Адресная доставка сообщений пользователям на разных узлах
- Предотвращение дублирования сообщений
- Компрессия данных для экономии полосы пропускания

```go
type PubSub struct {
    redisClient *redis.Client
    channelName string
    messageHandler func([]byte)
    pubsubConn *redis.PubSub
    instanceID string
    ctx context.Context
    cancel context.CancelFunc
}
```

## Шардирование

Шардирование - ключевой механизм для масштабирования WebSocket соединений. Основные особенности:

### Распределение клиентов

Клиенты распределяются по шардам с использованием одного из алгоритмов:
- Хеширование по ID пользователя (по умолчанию)
- Round-robin для равномерного распределения
- Адаптивное распределение на основе нагрузки шардов

```go
func (sh *ShardedHub) getShardForClient(client *Client) *Hub {
    if sh.Options.ShardingAlgorithm == "user_id" && client.UserID > 0 {
        // Распределение на основе UserID
        shardIndex := int(client.UserID % uint(sh.ShardCount))
        return sh.Shards[shardIndex]
    } else if sh.Options.ShardingAlgorithm == "adaptive" {
        // Адаптивное распределение на основе нагрузки
        return sh.getLeastLoadedShard()
    } else {
        // Round-robin по умолчанию
        sh.Mutex.Lock()
        defer sh.Mutex.Unlock()
        
        currShard := sh.nextShardIndex
        sh.nextShardIndex = (sh.nextShardIndex + 1) % sh.ShardCount
        return sh.Shards[currShard]
    }
}
```

### Балансировка нагрузки

Подсистема поддерживает динамическую балансировку нагрузки:
- Периодическая проверка распределения клиентов
- Выявление "горячих" шардов с высокой нагрузкой
- Миграция клиентов из перегруженных шардов
- Предупреждения о дисбалансе через метрики

```go
func (sh *ShardedHub) balanceShards() {
    // Находим самый нагруженный и самый ненагруженный шарды
    maxLoad := 0
    minLoad := math.MaxInt32
    maxShardIndex := 0
    minShardIndex := 0
    
    for i, hub := range sh.Shards {
        load := hub.ClientCount()
        if load > maxLoad {
            maxLoad = load
            maxShardIndex = i
        }
        if load < minLoad {
            minLoad = load
            minShardIndex = i
        }
    }
    
    // Проверяем дисбаланс
    if maxLoad - minLoad > sh.Options.RebalanceThreshold {
        // Мигрируем клиентов
        sh.migrateClients(maxShardIndex, minShardIndex, (maxLoad - minLoad) / 2)
    }
}
```

### Оптимизация доставки сообщений

Шардирование оптимизирует доставку сообщений:
- Локальные сообщения обрабатываются в рамках шарда
- Широковещательные сообщения параллельно обрабатываются в каждом шарде
- Адресная доставка направляется в конкретный шард
- Оптимизация для шаблонов соединений (группировка по UserID)

## Кластеризация

Для поддержки горизонтального масштабирования используется кластерный режим:

### Redis PubSub для синхронизации

Redis Pub/Sub обеспечивает обмен сообщениями между узлами:
- Каждый экземпляр сервера подписывается на общий канал
- При широковещательной рассылке сообщение публикуется в Redis
- Все узлы получают сообщение и доставляют его своим клиентам
- Для целевых сообщений используются каналы с идентификаторами пользователей

```go
func (ps *PubSub) PublishMessage(message []byte) error {
    return ps.redisClient.Publish(ps.ctx, ps.channelName, message).Err()
}

func (ps *PubSub) startSubscriber() {
    ps.pubsubConn = ps.redisClient.Subscribe(ps.ctx, ps.channelName)
    
    for {
        msg, err := ps.pubsubConn.ReceiveMessage(ps.ctx)
        if err != nil {
            log.Printf("Error receiving pubsub message: %v", err)
            continue
        }
        
        // Вызов обработчика сообщения
        ps.messageHandler([]byte(msg.Payload))
    }
}
```

### Предотвращение дублирования

Для предотвращения дублирования сообщений используется:
- Уникальный ID для каждого экземпляра сервера
- Маркировка сообщений ID отправителя
- Фильтрация собственных сообщений при получении
- Дедупликация по идентификатору сообщения

```go
type ClusteredMessage struct {
    OriginInstanceID string    `json:"origin_id"`
    MessageID        string    `json:"msg_id"`
    Timestamp        int64     `json:"ts"`
    Payload          []byte    `json:"payload"`
}
```

### Масштабирование и обнаружение

Кластерная архитектура поддерживает:
- Динамическое добавление/удаление узлов
- Обнаружение новых узлов через Redis
- Балансировку нагрузки между узлами
- Аварийное восстановление при отказе узла

## Оптимизация производительности

### Пул рабочих потоков

Для эффективной обработки сообщений используется пул воркеров:
- Фиксированное количество горутин для обработки сообщений
- Очередь заданий с приоритизацией
- Распараллеливание обработки для разных типов сообщений
- Предотвращение конкуренции за ресурсы

```go
type WorkerPool struct {
    Tasks       chan Task
    WorkerCount int
    wg          sync.WaitGroup
    quit        chan struct{}
}

type Task struct {
    Handler func() error
    Priority int
}
```

### Приоритизация сообщений

Реализована система приоритетов для сообщений:
- CRITICAL (3) - критичные системные сообщения (миграция шардов, ротация ключей)
- HIGH (2) - важные сообщения (начало/конец викторины, новый вопрос)
- NORMAL (1) - стандартные сообщения (обновление результатов)
- LOW (0) - низкоприоритетные сообщения (heartbeat, статистика)

```go
func (m *Manager) SendPrioritizedMessage(client *Client, message []byte, priority int) {
    task := Task{
        Handler: func() error {
            return client.Send(message)
        },
        Priority: priority,
    }
    
    m.workerPool.AddTask(task)
}
```

### Буферизация и пакетная обработка

Для оптимизации производительности используются:
- Буферизация исходящих сообщений
- Пакетная обработка однотипных сообщений
- Компрессия повторяющихся данных
- Отложенная отправка низкоприоритетных сообщений

```go
func (c *Client) batchSend(force bool) {
    c.sendMutex.Lock()
    defer c.sendMutex.Unlock()
    
    // Если буфер достаточно заполнен или принудительная отправка
    if len(c.sendBuffer) > c.batchThreshold || force {
        c.Conn.WriteMessage(websocket.TextMessage, c.sendBuffer.Bytes())
        c.sendBuffer.Reset()
    }
}
```

## Мониторинг и метрики

### Сбор и предоставление метрик

Подсистема предоставляет богатый набор метрик:
- Количество активных подключений (всего и по шардам)
- Скорость подключения/отключения клиентов
- Объем входящего/исходящего трафика
- Время обработки сообщений
- Распределение нагрузки по шардам
- Очереди сообщений и их состояние

```go
type Metrics struct {
    ActiveConnections        int64
    ConnectionsPerShard      map[int]int
    MessagesSent             int64
    MessagesReceived         int64
    AverageProcessingTimeMs  float64
    ErrorCount               int64
    LastErrorTime            time.Time
    LastError                string
    StartTime                time.Time
    mutex                    sync.RWMutex
}

func (m *Metrics) ToJSON() ([]byte, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    return json.Marshal(m)
}
```

### Интеграция с системами мониторинга

Поддерживается экспорт метрик в различные форматы:
- JSON для внутреннего использования
- Prometheus-совместимый формат
- Logstash/ELK-совместимые логи
- Графические дашборды через Grafana

### Алерты и уведомления

Система поддерживает автоматические алерты:
- Предупреждения о высокой нагрузке на шарды
- Оповещения о чрезмерной фрагментации
- Уведомления о слишком большом количестве ошибок
- Предупреждения о задержках в обработке сообщений

## Обработка ошибок и восстановление

### Устойчивость к отказам

Подсистема обеспечивает высокую отказоустойчивость:
- Изолированные шарды для предотвращения каскадных отказов
- Автоматическое восстановление после сбоев
- Перебалансировка при отказе шарда
- Корректное завершение при системных сбоях

### Обработка массовых отключений

Специальный механизм для обработки массовых отключений:
- Детектирование всплесков отключений
- Очереди обработки отключений для предотвращения блокировки
- Приоритизация критичных операций
- Постепенное освобождение ресурсов

```go
func (h *Hub) handleMassDisconnect(clients []*Client) {
    batchSize := 100
    for i := 0; i < len(clients); i += batchSize {
        end := i + batchSize
        if end > len(clients) {
            end = len(clients)
        }
        
        batch := clients[i:end]
        go func(clientBatch []*Client) {
            for _, client := range clientBatch {
                h.unregisterClient(client)
            }
        }(batch)
        
        // Небольшая пауза между пакетами для снижения нагрузки
        time.Sleep(50 * time.Millisecond)
    }
}
```

## API и интеграция

### WebSocket API

Основной WebSocket endpoint:
- `GET /ws` - подключение WebSocket с аутентификацией через JWT
- Закрытие соединения с кодами состояния
- Протокол на основе JSON для обмена сообщениями
- Настраиваемые таймауты и буферы

```go
func (h *WSHandler) ServeWS(c *gin.Context) {
    // Получение и проверка токена
    userID, err := h.authenticateRequest(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }
    
    // Обновление заголовков для WebSocket
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            // Проверка разрешенных источников
            return true // В реальности здесь проверка CORS
        },
    }
    
    // Обновление HTTP соединения до WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("Error upgrading connection: %v", err)
        return
    }
    
    // Создание и регистрация клиента
    client := NewClient(conn, userID)
    h.wsManager.RegisterClient(client)
    
    // Запуск горутин для чтения/записи
    go client.readPump()
    go client.writePump()
}
```

### HTTP API для мониторинга

Доступны HTTP эндпоинты для мониторинга и управления:
- `GET /api/ws/metrics` - получение метрик в JSON
- `GET /api/ws/metrics/prometheus` - метрики в формате Prometheus
- `GET /api/ws/health` - проверка работоспособности
- `GET /api/ws/shards` - информация о состоянии шардов
- `GET /api/ws/connections` - информация о подключениях

### Примеры интеграции

#### Клиентский код (JavaScript)

```javascript
// Создание WebSocket соединения с токеном аутентификации
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
const ws = new WebSocket(`wss://api.example.com/ws?token=${token}`);

// Обработчики событий
ws.onopen = () => {
  console.log('WebSocket соединение установлено');
  
  // Отправка сообщения готовности
  ws.send(JSON.stringify({
    type: 'USER_READY',
    data: { device_info: navigator.userAgent }
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  // Обработка разных типов сообщений
  switch (message.type) {
    case 'QUIZ_START':
      handleQuizStart(message.data);
      break;
    case 'QUESTION_START':
      handleNewQuestion(message.data);
      break;
    case 'RESULT_UPDATE':
      updateResults(message.data);
      break;
    // Другие типы сообщений...
  }
};

// Отправка ответа на вопрос
function sendAnswer(questionId, optionId) {
  ws.send(JSON.stringify({
    type: 'USER_ANSWER',
    data: {
      question_id: questionId,
      option_id: optionId,
      answer_time: new Date().toISOString()
    }
  }));
}

// Периодический ping для поддержания соединения
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'USER_HEARTBEAT' }));
  }
}, 30000);
```

#### Серверная интеграция (Go)

```go
// Инициализация WebSocket менеджера
wsManager := websocket.NewManager(redisClient, wsConfig)

// Регистрация обработчиков сообщений
wsManager.RegisterHandler("USER_ANSWER", func(client *websocket.Client, data json.RawMessage) error {
    var answer UserAnswerEvent
    if err := json.Unmarshal(data, &answer); err != nil {
        return err
    }
    
    // Обработка ответа пользователя
    result := quizManager.ProcessUserAnswer(client.UserID, answer.QuestionID, answer.OptionID)
    
    // Отправка результата обратно пользователю
    responseMsg := ResultMessage{
        Type: "ANSWER_RESULT",
        Data: result,
    }
    
    responseJSON, _ := json.Marshal(responseMsg)
    return client.Send(responseJSON)
})

// Широковещательная рассылка нового вопроса
func broadcastNewQuestion(quizID uint, question QuestionData) {
    message := WebSocketMessage{
        Type: "QUESTION_START",
        Data: question,
    }
    
    messageJSON, _ := json.Marshal(message)
    wsManager.BroadcastToQuiz(quizID, messageJSON, websocket.PriorityHigh)
}

// Интеграция с HTTP роутером
router.GET("/ws", wsHandler.ServeWS)
router.GET("/api/ws/metrics", wsHandler.GetMetrics)
router.GET("/api/ws/health", wsHandler.CheckHealth)
```

## Рекомендации по масштабированию

### Параметры производительности

Рекомендуемые настройки для разных сценариев:

| Параметр | Малая нагрузка | Средняя нагрузка | Высокая нагрузка |
|----------|---------------|-----------------|-----------------|
| Шарды | 2-4 | 8-16 | 32-64 |
| Клиентов на шард | 1000 | 2000 | 5000 |
| Воркеры | 4-8 | 16-32 | 64-128 |
| Размер буфера сообщений | 256 | 512 | 1024 |
| Интервал пинга (сек) | 60 | 30 | 15 |
| Redis соединения | 5 | 10-20 | 20-50 |

### Горизонтальное масштабирование

Рекомендации по горизонтальному масштабированию:
1. Включите режим кластеризации через Redis
2. Используйте балансировщик нагрузки с sticky-sessions
3. Настройте префиксы для Redis ключей для изоляции между экземплярами
4. Мониторите задержки между Redis и серверами
5. Настройте число шардов пропорционально доступной памяти 