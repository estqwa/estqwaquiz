package websocket

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ShardedHubConfig содержит настройки для ShardedHub
type ShardedHubConfig struct {
	// Количество шардов
	ShardCount int

	// Максимальное количество клиентов в одном шарде
	MaxClientsPerShard int

	// Конфигурация кластера
	ClusterConfig ClusterConfig
}

// DefaultShardedHubConfig возвращает конфигурацию по умолчанию
func DefaultShardedHubConfig() ShardedHubConfig {
	return ShardedHubConfig{
		ShardCount:         8,    // По умолчанию 8 шардов
		MaxClientsPerShard: 2000, // По умолчанию 2000 клиентов на шард
		ClusterConfig:      DefaultClusterConfig(),
	}
}

// WorkerPool представляет пул воркеров для обработки сообщений
type WorkerPool struct {
	tasks        chan func()
	workerCount  int
	wg           sync.WaitGroup
	shuttingDown int32 // атомарный флаг для отслеживания состояния завершения
}

// NewWorkerPool создает новый пул воркеров с указанным количеством
func NewWorkerPool(workerCount int) *WorkerPool {
	// Минимальное количество воркеров
	if workerCount < 1 {
		workerCount = 1
	}

	// Размер буфера задач - в 10 раз больше количества воркеров
	// для обеспечения непрерывной обработки
	pool := &WorkerPool{
		tasks:       make(chan func(), workerCount*10),
		workerCount: workerCount,
	}

	pool.Start()
	return pool
}

// Start запускает всех воркеров в пуле
func (p *WorkerPool) Start() {
	// Завершаем отложенные задачи при повторном запуске
	select {
	case <-p.tasks:
	default:
	}

	atomic.StoreInt32(&p.shuttingDown, 0)

	p.wg.Add(p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		go p.worker(i)
	}

	log.Printf("WorkerPool: запущен пул с %d воркерами", p.workerCount)
}

// worker запускает цикл обработки задач
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	log.Printf("WorkerPool: воркер %d запущен", id)

	for task := range p.tasks {
		// Проверяем, не завершается ли пул
		if atomic.LoadInt32(&p.shuttingDown) == 1 {
			log.Printf("WorkerPool: воркер %d завершает работу при закрытии пула", id)
			return
		}

		// Выполняем задачу с защитой от паники
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("WorkerPool: воркер %d восстановился после паники: %v", id, r)
				}
			}()

			task()
		}()
	}

	log.Printf("WorkerPool: воркер %d завершил работу", id)
}

// Submit добавляет задачу в пул на выполнение
func (p *WorkerPool) Submit(task func()) bool {
	// Проверяем, не завершается ли пул
	if atomic.LoadInt32(&p.shuttingDown) == 1 {
		return false
	}

	select {
	case p.tasks <- task:
		return true
	default:
		// Если буфер переполнен, возвращаем false
		return false
	}
}

// Stop останавливает все воркеры и ожидает их завершения
func (p *WorkerPool) Stop() {
	atomic.StoreInt32(&p.shuttingDown, 1)
	close(p.tasks)
	p.wg.Wait()
	log.Printf("WorkerPool: пул остановлен, все воркеры завершили работу")
}

// ShardedHub представляет собой хаб с шардированием клиентов
// для эффективной обработки большого числа подключений
type ShardedHub struct {
	// Шарды для распределения клиентов
	shards []*Shard

	// Количество шардов
	shardCount int

	// Максимальное количество клиентов в шарде
	maxClientsPerShard int

	// Менеджер метрик
	metrics *HubMetrics

	// Компонент для межсерверного взаимодействия
	cluster *ClusterHub

	// Канал для завершения работы фоновых горутин
	done chan struct{}

	// Пул воркеров для обработки задач
	workerPool *WorkerPool

	// Каналы для алертинга
	alertChan chan AlertMessage

	// Функция для обработки алертов (может быть заменена пользователем)
	alertHandler func(AlertMessage)

	// Мьютекс для безопасной работы с alertHandler
	alertMu sync.RWMutex
}

// AlertType определяет тип алерта
type AlertType string

const (
	// AlertHotShard сигнализирует о "горячем" шарде
	AlertHotShard AlertType = "hot_shard"

	// AlertMessageLoss сигнализирует о потерянных сообщениях
	AlertMessageLoss AlertType = "message_loss"

	// AlertBufferOverflow сигнализирует о переполнении буфера
	AlertBufferOverflow AlertType = "buffer_overflow"

	// AlertHighLatency сигнализирует о высокой задержке обработки сообщений
	AlertHighLatency AlertType = "high_latency"
)

// AlertSeverity определяет уровень серьезности алерта
type AlertSeverity string

const (
	// AlertInfo информационный уровень
	AlertInfo AlertSeverity = "info"

	// AlertWarning уровень предупреждения
	AlertWarning AlertSeverity = "warning"

	// AlertCritical критический уровень
	AlertCritical AlertSeverity = "critical"
)

// AlertMessage представляет сообщение алерта
type AlertMessage struct {
	// Тип алерта
	Type AlertType `json:"type"`

	// Уровень серьезности
	Severity AlertSeverity `json:"severity"`

	// Сообщение
	Message string `json:"message"`

	// Метаданные алерта
	Metadata map[string]interface{} `json:"metadata"`

	// Время создания
	Timestamp time.Time `json:"timestamp"`
}

// Проверка компилятором, что ShardedHub реализует интерфейс HubInterface
var _ HubInterface = (*ShardedHub)(nil)

// NewShardedHub создает новый ShardedHub с указанной конфигурацией
func NewShardedHub(config ShardedHubConfig) *ShardedHub {
	if config.ShardCount <= 0 {
		config.ShardCount = 8
	}
	if config.MaxClientsPerShard <= 0 {
		config.MaxClientsPerShard = 2000
	}

	metrics := NewHubMetrics()

	// Создаем пул воркеров
	workerPool := NewWorkerPool(config.ShardCount * 2)
	workerPool.Start()

	hub := &ShardedHub{
		shardCount:         config.ShardCount,
		maxClientsPerShard: config.MaxClientsPerShard,
		metrics:            metrics,
		done:               make(chan struct{}),
		workerPool:         workerPool,
		alertChan:          make(chan AlertMessage, 100),
	}

	// Инициализируем обработчик алертов по умолчанию
	hub.alertHandler = hub.defaultAlertHandler

	// Создаем шарды
	hub.shards = make([]*Shard, hub.shardCount)
	for i := 0; i < hub.shardCount; i++ {
		hub.shards[i] = NewShard(i, hub, hub.maxClientsPerShard)
	}

	// Создаем компонент для кластерного режима
	hub.cluster = NewClusterHub(hub, config.ClusterConfig, metrics)

	return hub
}

// defaultAlertHandler обрабатывает алерты по умолчанию - просто логирует их
func (h *ShardedHub) defaultAlertHandler(alert AlertMessage) {
	switch alert.Severity {
	case AlertCritical:
		log.Printf("[КРИТИЧЕСКИЙ АЛЕРТ] %s: %s", alert.Type, alert.Message)
	case AlertWarning:
		log.Printf("[ПРЕДУПРЕЖДЕНИЕ] %s: %s", alert.Type, alert.Message)
	default:
		log.Printf("[ИНФО] %s: %s", alert.Type, alert.Message)
	}

	// Логируем метаданные для отладки
	metadataJson, _ := json.Marshal(alert.Metadata)
	log.Printf("[АЛЕРТ ДЕТАЛИ] %s", string(metadataJson))
}

// SetAlertHandler устанавливает пользовательский обработчик алертов
func (h *ShardedHub) SetAlertHandler(handler func(AlertMessage)) {
	h.alertMu.Lock()
	defer h.alertMu.Unlock()
	h.alertHandler = handler
}

// SendAlert отправляет алерт
func (h *ShardedHub) SendAlert(alertType AlertType, severity AlertSeverity, message string, metadata map[string]interface{}) {
	alert := AlertMessage{
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	// Отправляем неблокирующим способом
	select {
	case h.alertChan <- alert:
		// Успешно отправлено
	default:
		// Буфер алертов переполнен, логируем это напрямую
		log.Printf("[ПЕРЕПОЛНЕНИЕ БУФЕРА АЛЕРТОВ] %s: %s", alertType, message)
	}
}

// Run запускает все шарды и кластерный компонент
func (h *ShardedHub) Run() {
	log.Printf("ShardedHub: запуск с %d шардами, до %d клиентов на шард",
		h.shardCount, h.maxClientsPerShard)

	// Запускаем все шарды
	for _, shard := range h.shards {
		go shard.Run()
	}

	// Запускаем сбор метрик
	go h.collectMetrics()

	// Запускаем автоматический балансировщик шардов
	go h.RunBalancer()

	// Запускаем кластерный компонент
	if err := h.cluster.Start(); err != nil {
		log.Printf("ShardedHub: ошибка запуска кластерного компонента: %v", err)
	}

	// Запускаем обработчик алертов
	go h.handleAlerts()

	// Ожидаем сигнал завершения работы
	<-h.done
	log.Println("ShardedHub: завершение работы")
}

// getShardID вычисляет ID шарда для указанного userID
func (h *ShardedHub) getShardID(userID string) int {
	if userID == "" {
		// Для пустых userID используем псевдослучайное значение на основе времени
		// вместо всегда последнего шарда, чтобы избежать его перегрузки
		now := time.Now().UnixNano()
		return int(now % int64(h.shardCount))
	}

	// Используем хеш-функцию для равномерного распределения
	hasher := fnv.New32a()
	hasher.Write([]byte(userID))
	return int(hasher.Sum32() % uint32(h.shardCount))
}

// getShard возвращает шард для указанного userID
func (h *ShardedHub) getShard(userID string) *Shard {
	shardID := h.getShardID(userID)
	return h.shards[shardID]
}

// RegisterClient регистрирует клиента в соответствующем шарде
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) RegisterClient(client *Client) {
	shard := h.getShard(client.UserID)
	shard.register <- client
}

// RegisterSync регистрирует клиента и ожидает завершения регистрации
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) RegisterSync(client *Client, done chan struct{}) {
	// Добавляем канал обратного вызова в клиента
	client.registrationComplete = done

	// Регистрируем клиента в соответствующем шарде
	shard := h.getShard(client.UserID)
	shard.register <- client
}

// UnregisterClient отменяет регистрацию клиента
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) UnregisterClient(client *Client) {
	shard := h.getShard(client.UserID)
	shard.unregister <- client
}

// Broadcast отправляет сообщение всем клиентам
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) Broadcast(message []byte) {
	h.BroadcastBytes(message)
}

// BroadcastBytes отправляет байтовое сообщение всем клиентам
func (h *ShardedHub) BroadcastBytes(message []byte) {
	// Параллельная рассылка по всем шардам с использованием пула воркеров
	var wg sync.WaitGroup
	wg.Add(len(h.shards))

	for _, shard := range h.shards {
		// Используем пул воркеров для распределения нагрузки
		currentShard := shard // Создаем локальную копию для замыкания
		if !h.workerPool.Submit(func() {
			defer wg.Done()
			currentShard.BroadcastBytes(message)
		}) {
			// Если пул воркеров переполнен, выполняем задачу напрямую
			go func(s *Shard) {
				defer wg.Done()
				s.BroadcastBytes(message)
			}(shard)
		}
	}

	// Если включен кластерный режим, отправляем сообщение в другие экземпляры
	if h.cluster != nil {
		go h.cluster.BroadcastToCluster(message)
	}

	// Для обычных сообщений не ждем завершения отправки во все шарды
}

// BroadcastJSON отправляет JSON структуру всем клиентам
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) BroadcastJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	h.BroadcastBytes(data)
	return nil
}

// SendToUser отправляет сообщение конкретному пользователю
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) SendToUser(userID string, message []byte) bool {
	shard := h.getShard(userID)
	result := shard.SendToUser(userID, message)

	// Если пользователь не найден в локальном экземпляре,
	// пробуем отправить через кластер
	if !result && h.cluster != nil {
		go func() {
			if err := h.cluster.SendToUserInCluster(userID, message); err != nil {
				log.Printf("ShardedHub: ошибка отправки сообщения пользователю %s через кластер: %v",
					userID, err)
			}
		}()
	}

	return result
}

// SendJSONToUser отправляет JSON структуру конкретному пользователю
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) SendJSONToUser(userID string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	h.SendToUser(userID, data)
	return nil
}

// ClientCount возвращает общее количество подключенных клиентов
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) ClientCount() int {
	var count int
	for _, shard := range h.shards {
		count += shard.GetClientCount()
	}
	return count
}

// GetMetrics возвращает метрики хаба
// Совместимость с интерфейсом старого Hub
func (h *ShardedHub) GetMetrics() map[string]interface{} {
	return h.metrics.GetBasicMetrics()
}

// GetDetailedMetrics возвращает детальные метрики хаба
func (h *ShardedHub) GetDetailedMetrics() map[string]interface{} {
	return h.metrics.GetAllMetrics()
}

// GetLeastLoadedShardID возвращает ID наименее загруженного шарда
func (h *ShardedHub) GetLeastLoadedShardID() int {
	leastLoaded := 0
	minClients := h.shards[0].GetClientCount()

	for i := 1; i < len(h.shards); i++ {
		count := h.shards[i].GetClientCount()
		if count < minClients {
			minClients = count
			leastLoaded = i
		}
	}

	return leastLoaded
}

// GetMostLoadedShardID возвращает ID наиболее загруженного шарда
func (h *ShardedHub) GetMostLoadedShardID() int {
	mostLoaded := 0
	maxClients := h.shards[0].GetClientCount()

	for i := 1; i < len(h.shards); i++ {
		count := h.shards[i].GetClientCount()
		if count > maxClients {
			maxClients = count
			mostLoaded = i
		}
	}

	return mostLoaded
}

// IsShardBalancingNeeded проверяет, нужно ли перебалансирование шардов
func (h *ShardedHub) IsShardBalancingNeeded() bool {
	mostLoadedID := h.GetMostLoadedShardID()
	leastLoadedID := h.GetLeastLoadedShardID()

	mostLoadedCount := h.shards[mostLoadedID].GetClientCount()
	leastLoadedCount := h.shards[leastLoadedID].GetClientCount()

	// Если разница больше 30% и в перегруженном шарде больше определенного порога клиентов,
	// рекомендуется перебалансирование
	threshold := 100 // Минимальное количество клиентов для запуска ребалансировки
	if mostLoadedCount > threshold && leastLoadedCount > 0 {
		imbalanceRatio := float64(mostLoadedCount) / float64(leastLoadedCount)
		return imbalanceRatio > 1.3 // 30% дисбаланс
	}

	return false
}

// collectMetrics периодически собирает метрики со всех шардов
func (h *ShardedHub) collectMetrics() {
	log.Println("ShardedHub: сбор метрик")

	// Создаем метрики для всех шардов
	shardMetrics := make([]map[string]interface{}, h.shardCount)
	hotShards := make([]int, 0)
	totalConnections := int64(0)
	maxLoad := float64(0)
	maxLoadShardID := -1

	// Собираем метрики со всех шардов
	for i, shard := range h.shards {
		metrics := shard.GetMetrics()
		shardMetrics[i] = metrics

		// Обновляем общее количество соединений
		if connections, ok := metrics["active_connections"].(int); ok {
			totalConnections += int64(connections)
		}

		// Проверяем нагрузку шарда
		if loadPercentage, ok := metrics["load_percentage"].(float64); ok {
			if loadPercentage > maxLoad {
				maxLoad = loadPercentage
				maxLoadShardID = i
			}

			// Определяем "горячие" шарды
			if loadPercentage > 75 {
				hotShards = append(hotShards, i)

				// Отправляем алерт для "горячего" шарда
				severity := AlertWarning
				if loadPercentage > 90 {
					severity = AlertCritical
				}

				h.SendAlert(AlertHotShard, severity,
					fmt.Sprintf("Обнаружен горячий шард %d с загрузкой %.2f%%", i, loadPercentage),
					map[string]interface{}{
						"shard_id":           i,
						"load_percentage":    loadPercentage,
						"active_connections": metrics["active_connections"],
						"max_clients":        metrics["max_clients"],
					})
			}

			// Проверяем статистику отключений, если доступна
			if disconnectionStats, ok := metrics["disconnection_stats"].(map[string]interface{}); ok {
				if bufferAlert, ok := disconnectionStats["buffer_alert_triggered"].(bool); ok && bufferAlert {
					h.SendAlert(AlertBufferOverflow, AlertCritical,
						fmt.Sprintf("Переполнение буфера отключений в шарде %d", i),
						map[string]interface{}{
							"shard_id":            i,
							"disconnection_stats": disconnectionStats,
						})
				}
			}
		}
	}

	// Обновляем метрики хаба
	h.metrics.mu.Lock()
	h.metrics.activeConnections = totalConnections
	h.metrics.UpdateShardMetrics(shardMetrics)
	h.metrics.mu.Unlock()

	// Проверяем, нужна ли балансировка
	if len(hotShards) > 0 {
		log.Printf("ShardedHub: обнаружены горячие шарды: %v", hotShards)

		// Отправляем общий алерт о "горячих" шардах
		h.SendAlert(AlertHotShard, AlertWarning,
			fmt.Sprintf("Обнаружено %d горячих шардов, максимальная нагрузка %.2f%% (шард %d)",
				len(hotShards), maxLoad, maxLoadShardID),
			map[string]interface{}{
				"hot_shards":        hotShards,
				"max_load":          maxLoad,
				"max_load_shard":    maxLoadShardID,
				"total_connections": totalConnections,
			})

		// Если нужна срочная балансировка, запускаем внеплановую
		if maxLoad > 95 && len(hotShards) > h.shardCount/4 {
			log.Printf("ShardedHub: инициирована экстренная балансировка шардов")

			// Отправляем задачу в пул воркеров
			h.workerPool.Submit(func() {
				migratedCount := h.BalanceShards()
				log.Printf("ShardedHub: экстренная балансировка завершена, мигрировано %d клиентов", migratedCount)
			})
		}
	}
}

// MigrateClientToShard перемещает клиента из одного шарда в другой
func (h *ShardedHub) MigrateClientToShard(client *Client, targetShardID int) bool {
	if targetShardID < 0 || targetShardID >= h.shardCount {
		log.Printf("ShardedHub: невозможно мигрировать клиента %s в недопустимый шард %d",
			client.UserID, targetShardID)
		return false
	}

	currentShardID := h.getShardID(client.UserID)
	if currentShardID == targetShardID {
		// Клиент уже в целевом шарде
		return true
	}

	log.Printf("ShardedHub: начинаем миграцию клиента %s из шарда %d в шард %d",
		client.UserID, currentShardID, targetShardID)

	// Создаем временную копию информации о клиенте
	// для повторной регистрации в новом шарде
	newClient := &Client{
		UserID:               client.UserID,
		ConnectionID:         client.ConnectionID,
		conn:                 client.conn,
		send:                 make(chan []byte, len(client.send)), // Создаем новый буфер
		lastActivity:         client.lastActivity,
		registrationComplete: make(chan struct{}, 1),
	}

	// Копируем роли клиента
	newClient.roles = make(map[string]bool)
	client.subMutex.RLock()
	for role, hasRole := range client.roles {
		if hasRole {
			newClient.roles[role] = true
		}
	}
	client.subMutex.RUnlock()

	// Копируем подписки клиента
	newClient.subscriptions = sync.Map{}
	client.subscriptions.Range(func(key, value interface{}) bool {
		if msgType, ok := key.(string); ok {
			newClient.subscriptions.Store(msgType, true)
		}
		return true
	})

	// Копируем непрочитанные сообщения из старого буфера
	pending := len(client.send)
	for i := 0; i < pending; i++ {
		select {
		case msg := <-client.send:
			newClient.send <- msg
		default:
			// Ничего не делаем, выходим из select
		}
	}

	// Отправляем информационное сообщение клиенту о миграции
	infoMsg := map[string]interface{}{
		"type": "system",
		"data": map[string]interface{}{
			"event": "shard_migration",
			"from":  currentShardID,
			"to":    targetShardID,
		},
	}

	if infoData, err := json.Marshal(infoMsg); err == nil {
		select {
		case newClient.send <- infoData:
			log.Printf("ShardedHub: отправлено уведомление о миграции клиенту %s", client.UserID)
		default:
			log.Printf("ShardedHub: не удалось отправить уведомление о миграции клиенту %s", client.UserID)
		}
	}

	// Регистрируем клиента в новом шарде
	h.shards[targetShardID].register <- newClient

	// Ожидаем завершения регистрации в новом шарде с таймаутом
	migrationTimeout := 5 * time.Second
	migrationSuccess := false

	select {
	case <-newClient.registrationComplete:
		log.Printf("ShardedHub: клиент %s успешно зарегистрирован в новом шарде %d",
			client.UserID, targetShardID)
		migrationSuccess = true
	case <-time.After(migrationTimeout):
		log.Printf("ShardedHub: таймаут при миграции клиента %s в шард %d",
			client.UserID, targetShardID)

		// Отправляем алерт о проблеме миграции
		h.SendAlert(
			"migration_failure",
			"warning",
			fmt.Sprintf("Таймаут миграции клиента %s в шард %d", client.UserID, targetShardID),
			map[string]interface{}{
				"user_id":          client.UserID,
				"connection_id":    client.ConnectionID,
				"from_shard":       currentShardID,
				"to_shard":         targetShardID,
				"migration_status": "timeout",
			},
		)

		// Возвращаем false - миграция не удалась
		return false
	}

	// Отменяем регистрацию в старом шарде только при успешной миграции
	if migrationSuccess {
		h.shards[currentShardID].unregister <- client

		// Отправляем метрику успешной миграции
		h.metrics.mu.Lock()
		if _, ok := h.metrics.messageTypeCounts["shard_migration_success"]; ok {
			h.metrics.messageTypeCounts["shard_migration_success"]++
		} else {
			h.metrics.messageTypeCounts["shard_migration_success"] = 1
		}
		h.metrics.mu.Unlock()

		log.Printf("ShardedHub: миграция клиента %s успешно завершена", client.UserID)

		// Отправляем алерт о успешной миграции (при балансировке шардов)
		h.SendAlert(
			"migration_success",
			"info",
			fmt.Sprintf("Успешная миграция клиента %s в шард %d", client.UserID, targetShardID),
			map[string]interface{}{
				"user_id":             client.UserID,
				"connection_id":       client.ConnectionID,
				"from_shard":          currentShardID,
				"to_shard":            targetShardID,
				"migration_status":    "success",
				"subscriptions_count": len(newClient.GetSubscriptions()),
				"roles_count":         len(newClient.roles),
			},
		)

		return true
	}

	// Если миграция не удалась, но клиент уже зарегистрирован в новом шарде,
	// нужно отменить регистрацию в обоих шардах
	h.shards[currentShardID].unregister <- client
	h.shards[targetShardID].unregister <- newClient

	return false
}

// BalanceShards выполняет автоматическое перебалансирование шардов
func (h *ShardedHub) BalanceShards() int {
	if !h.IsShardBalancingNeeded() {
		return 0
	}

	log.Printf("ShardedHub: начинаем автоматическое перебалансирование шардов")

	mostLoadedID := h.GetMostLoadedShardID()
	leastLoadedID := h.GetLeastLoadedShardID()

	mostLoadedShard := h.shards[mostLoadedID]
	leastLoadedShard := h.shards[leastLoadedID]

	// Определяем количество клиентов для миграции
	mostLoadedCount := mostLoadedShard.GetClientCount()
	leastLoadedCount := leastLoadedShard.GetClientCount()

	// Вычисляем целевое значение для равномерности
	targetCount := (mostLoadedCount + leastLoadedCount) / 2

	// Сколько клиентов нужно переместить
	clientsToMove := mostLoadedCount - targetCount

	log.Printf("ShardedHub: перемещаем %d клиентов из шарда %d в шард %d",
		clientsToMove, mostLoadedID, leastLoadedID)

	// Ограничиваем количество миграций за один проход
	if clientsToMove > 50 {
		clientsToMove = 50
	}

	// Собираем клиентов для миграции
	clientsToMigrate := make([]*Client, 0, clientsToMove)

	// Используем mutex для безопасного доступа к slice при итерации по sync.Map
	var migrateMutex sync.Mutex

	// Собираем клиентов из наиболее загруженного шарда
	mostLoadedShard.clients.Range(func(key, value interface{}) bool {
		client, ok := key.(*Client)
		if !ok {
			return true // Пропускаем некорректные записи
		}

		migrateMutex.Lock()
		if len(clientsToMigrate) < clientsToMove {
			clientsToMigrate = append(clientsToMigrate, client)
		}
		migrateMutex.Unlock()

		// Прекращаем итерацию, если собрали достаточно клиентов
		return len(clientsToMigrate) < clientsToMove
	})

	// Мигрируем собранных клиентов
	migratedCount := 0
	for _, client := range clientsToMigrate {
		if h.MigrateClientToShard(client, leastLoadedID) {
			migratedCount++
		}
	}

	log.Printf("ShardedHub: перебалансирование завершено, успешно перемещено %d клиентов", migratedCount)
	return migratedCount
}

// RunBalancer запускает периодическое автоматическое перебалансирование
func (h *ShardedHub) RunBalancer() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Printf("ShardedHub: запущен автоматический балансировщик шардов")

	for {
		select {
		case <-h.done:
			log.Printf("ShardedHub: балансировщик шардов остановлен")
			return
		case <-ticker.C:
			h.BalanceShards()
		}
	}
}

// Close закрывает все шарды и освобождает ресурсы
func (h *ShardedHub) Close() {
	log.Println("ShardedHub: закрытие всех шардов")

	// Закрываем кластерный компонент
	if h.cluster != nil {
		h.cluster.Stop()
	}

	// Закрываем все шарды
	for _, shard := range h.shards {
		shard.Close()
	}

	// Закрываем пул воркеров
	if h.workerPool != nil {
		h.workerPool.Stop()
	}

	// Сигнал для завершения фоновых горутин
	close(h.done)

	log.Println("ShardedHub: все ресурсы освобождены")
}

// BroadcastPrioritized отправляет высокоприоритетное сообщение всем клиентам
// с дополнительными гарантиями доставки
func (h *ShardedHub) BroadcastPrioritized(message []byte) error {
	log.Printf("ShardedHub: рассылка высокоприоритетного сообщения")

	// Создаем WaitGroup для ожидания завершения отправки во все шарды
	var wg sync.WaitGroup
	wg.Add(len(h.shards))

	// Увеличенные буферы для высокоприоритетных сообщений
	// чтобы гарантировать, что они не будут отброшены
	for _, shard := range h.shards {
		// Используем пул воркеров для распределения нагрузки
		currentShard := shard // Создаем локальную копию для замыкания
		if !h.workerPool.Submit(func() {
			defer wg.Done()

			// Для высокоприоритетных сообщений блокируем отправку,
			// чтобы гарантировать доставку
			select {
			case currentShard.broadcast <- message:
				// Сообщение успешно отправлено в канал рассылки
			case <-time.After(1 * time.Second):
				// Если канал полный, обрабатываем сообщение напрямую
				log.Printf("Shard %d: приоритетная отправка через прямую рассылку", currentShard.id)

				// Подсчитываем клиентов для метрик
				var clientCount int

				// Выполняем прямую рассылку клиентам
				currentShard.clients.Range(func(key, value interface{}) bool {
					client, ok := key.(*Client)
					if !ok {
						return true // Пропускаем некорректные записи
					}

					// Блокирующая отправка с таймаутом
					select {
					case client.send <- message:
						clientCount++
					case <-time.After(500 * time.Millisecond):
						// Если буфер клиента переполнен и не освобождается, обрабатываем ошибку
						log.Printf("Shard %d: не удалось отправить приоритетное сообщение клиенту %s",
							currentShard.id, client.UserID)
					}

					return true
				})

				// Обновляем метрики
				if clientCount > 0 {
					currentShard.metrics.mu.Lock()
					currentShard.metrics.messagesSent += int64(clientCount)
					currentShard.metrics.mu.Unlock()

					log.Printf("Shard %d: приоритетное сообщение отправлено %d клиентам напрямую",
						currentShard.id, clientCount)
				}
			}
		}) {
			// Если пул воркеров переполнен, выполняем задачу напрямую
			log.Printf("ShardedHub: пул воркеров переполнен, выполняем задачу напрямую для шарда %d", shard.id)
			go func(s *Shard) {
				defer wg.Done()
				s.BroadcastBytes(message)
			}(shard)
		}
	}

	// Ожидаем завершения отправки во все шарды
	wg.Wait()

	// Если включен кластерный режим, отправляем сообщение в другие экземпляры
	if h.cluster != nil {
		go h.cluster.BroadcastToCluster(message)
	}

	return nil
}

// handleAlerts обрабатывает алерты
func (h *ShardedHub) handleAlerts() {
	for {
		select {
		case alert := <-h.alertChan:
			h.alertMu.RLock()
			handler := h.alertHandler
			h.alertMu.RUnlock()

			if handler != nil {
				handler(alert)
			}
		case <-h.done:
			return
		}
	}
}
