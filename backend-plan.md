# План разработки бэкенда для приложения викторины

## 1. Бизнес-логика викторины

### 1.1. Основной поток процессов

1. **Регистрация и аутентификация**
   - Пользователь регистрируется/входит в систему
   - Получает JWT токен для дальнейших запросов

2. **Подготовка к викторине**
   - Администратор настраивает время начала викторины
   - Пользователи видят обратный отсчет до начала
   - За несколько минут до старта открывается "зал ожидания"

3. **Проведение викторины**
   - В назначенное время автоматически начинается викторина
   - Все участники получают вопросы одновременно
   - На каждый вопрос дается ограниченное время (10-20 секунд)
   - После каждого вопроса показывается правильный ответ
   - Между вопросами короткая пауза (3-5 секунд)

4. **Обработка результатов**
   - Система подсчитывает количество правильных ответов
   - Учитывается скорость ответа (быстрее = больше очков)
   - Формируется рейтинг участников
   - Отображаются итоговые результаты

### 1.2. Ключевые бизнес-требования

- Одна викторина, запускаемая в назначенное время
- Синхронизация вопросов для всех пользователей
- Контроль времени ответа
- Невозможность изменить ответ после отправки
- Невозможность ответить после истечения времени

## 2. Структура проекта (Golang)

```
/trivia-api
│
├── cmd/                      # Точки входа приложения
│   ├── api/                  # Основное API приложение
│   │    └── main.go           # Основной файл запуска
│   └── tools/                # Вспомогательные инструменты
│       └── create_admin.go   # Создание администратора
│
├── internal/                 # Внутренний код приложения
│   ├── config/               # Конфигурация приложения
│   │   └── config.go
│   │
│   ├── domain/               # Бизнес-сущности и интерфейсы
│   │   ├── entity/           # Модели данных
│   │   │   ├── user.go
│   │   │   ├── quiz.go
│   │   │   ├── question.go
│   │   │   └── result.go
│   │   │
│   │   └── repository/       # Интерфейсы репозиториев
│   │       ├── user_repo.go
│   │       ├── quiz_repo.go
│   │       └── result_repo.go
│   │
│   ├── service/              # Бизнес-логика
│   │   ├── auth_service.go
│   │   ├── quiz_service.go
│   │   └── result_service.go
│   │
│   ├── handler/              # Обработчики запросов
│   │   ├── auth_handler.go
│   │   ├── quiz_handler.go
│   │   └── ws_handler.go
│   │
│   ├── middleware/           # Промежуточное ПО
│   │   ├── auth_middleware.go
│   │   └── logging_middleware.go
│   │
│   ├── repository/           # Реализации репозиториев
│   │   ├── postgres/
│   │   │   ├── user_repo.go
│   │   │   ├── quiz_repo.go
│   │   │   └── result_repo.go
│   │   │
│   │   └── redis/
│   │       └── cache_repo.go
│   │
│   └── websocket/            # WebSocket логика
│       ├── hub.go
│       ├── client.go
│       └── manager.go
│
├── pkg/                      # Переиспользуемые пакеты
│   ├── auth/                 # Аутентификация
│   │   └── jwt.go
│   │
│   ├── database/             # Работа с БД
│   │   ├── postgres.go
│   │   └── redis.go
│   │
│   └── logger/               # Логирование
│       └── logger.go
│
├── migrations/               # Миграции БД
│   └── ... 
│
├── scripts/                  # Скрипты для разработки
│   └── ...
│
└── docs/                     # Документация API
    └── swagger.yaml
```

## 3. Основные компоненты системы

### 3.1. Модели данных (domain/entity)

#### User (Пользователь)
```go
type User struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    Username       string    `gorm:"size:50;not null;unique" json:"username"`
    Email          string    `gorm:"size:100;not null;unique" json:"email"`
    Password       string    `gorm:"size:100;not null" json:"-"`
    ProfilePicture string    `gorm:"size:255" json:"profile_picture"`
    GamesPlayed    int       `json:"games_played"`
    TotalScore     int       `json:"total_score"`
    HighestScore   int       `json:"highest_score"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

#### Quiz (Викторина)
```go
type Quiz struct {
    ID              uint        `gorm:"primaryKey" json:"id"`
    Title           string      `gorm:"size:100;not null" json:"title"`
    Description     string      `gorm:"size:500" json:"description"`
    ScheduledTime   time.Time   `gorm:"not null" json:"scheduled_time"`
    Status          string      `gorm:"size:20;not null" json:"status"` // scheduled, in_progress, completed
    QuestionCount   int         `json:"question_count"`
    Questions       []Question  `gorm:"foreignKey:QuizID" json:"questions,omitempty"`
    CreatedAt       time.Time   `json:"created_at"`
    UpdatedAt       time.Time   `json:"updated_at"`
}
```

#### Question (Вопрос)
```go
type Question struct {
    ID              uint      `gorm:"primaryKey" json:"id"`
    QuizID          uint      `gorm:"not null" json:"quiz_id"`
    Text            string    `gorm:"size:500;not null" json:"text"`
    Options         []string  `gorm:"type:jsonb;not null" json:"options"`
    CorrectOption   int       `gorm:"not null" json:"-"` // Скрыто от клиента
    TimeLimitSec    int       `gorm:"not null" json:"time_limit_sec"`
    PointValue      int       `gorm:"not null" json:"point_value"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

#### UserAnswer (Ответ пользователя)
```go
type UserAnswer struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    UserID        uint      `gorm:"not null" json:"user_id"`
    QuizID        uint      `gorm:"not null" json:"quiz_id"`
    QuestionID    uint      `gorm:"not null" json:"question_id"`
    SelectedOption int       `json:"selected_option"`
    IsCorrect     bool      `json:"is_correct"`
    ResponseTimeMs int64     `json:"response_time_ms"`
    CreatedAt     time.Time `json:"created_at"`
}
```

#### Result (Результат)
```go
type Result struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    UserID         uint      `gorm:"not null" json:"user_id"`
    QuizID         uint      `gorm:"not null" json:"quiz_id"`
    Score          int       `gorm:"not null" json:"score"`
    CorrectAnswers int       `json:"correct_answers"`
    Rank           int       `json:"rank"`
    CreatedAt      time.Time `json:"created_at"`
}
```

### 3.2. WebSocket сообщения

#### Сообщения от сервера к клиенту

```go
// Старт обратного отсчета
type CountdownEvent struct {
    EventType string    `json:"type"`
    QuizID    uint      `json:"quiz_id"`
    StartTime time.Time `json:"start_time"`
    SecondsLeft int     `json:"seconds_left"`
}

// Новый вопрос
type QuestionEvent struct {
    EventType   string   `json:"type"`
    QuestionID  uint     `json:"question_id"`
    QuizID      uint     `json:"quiz_id"`
    Number      int      `json:"number"`
    Text        string   `json:"text"`
    Options     []string `json:"options"`
    TimeLimit   int      `json:"time_limit"`
    TotalQuestions int   `json:"total_questions"`
}

// Обновление таймера
type TimerEvent struct {
    EventType       string `json:"type"`
    QuestionID      uint   `json:"question_id"`
    RemainingSeconds int   `json:"remaining_seconds"`
}

// Результат ответа
type AnswerResultEvent struct {
    EventType     string `json:"type"`
    QuestionID    uint   `json:"question_id"`
    CorrectOption int    `json:"correct_option"`
    YourAnswer    int    `json:"your_answer"`
    IsCorrect     bool   `json:"is_correct"`
    PointsEarned  int    `json:"points_earned"`
    TotalScore    int    `json:"total_score"`
    TimeTakenMs   int64  `json:"time_taken_ms"`
}

// Конец викторины
type QuizEndEvent struct {
    EventType string    `json:"type"`
    QuizID    uint      `json:"quiz_id"`
    Results   []Result  `json:"results"`
}
```

#### Сообщения от клиента к серверу

```go
// Готовность к викторине
type ReadyEvent struct {
    EventType string `json:"type"`
    QuizID    uint   `json:"quiz_id"`
}

// Ответ на вопрос
type AnswerEvent struct {
    EventType      string `json:"type"`
    QuestionID     uint   `json:"question_id"`
    SelectedOption int    `json:"selected_option"`
    Timestamp      int64  `json:"timestamp"`
}
```

### 3.3. Основные интерфейсы репозиториев

```go
// UserRepository - интерфейс для работы с пользователями
type UserRepository interface {
    Create(user *entity.User) error
    GetByID(id uint) (*entity.User, error)
    GetByEmail(email string) (*entity.User, error)
    GetByUsername(username string) (*entity.User, error)
    Update(user *entity.User) error
    UpdateScore(userID uint, score int) error
}

// QuizRepository - интерфейс для работы с викторинами
type QuizRepository interface {
    Create(quiz *entity.Quiz) error
    GetByID(id uint) (*entity.Quiz, error)
    GetActive() (*entity.Quiz, error)
    GetWithQuestions(id uint) (*entity.Quiz, error)
    UpdateStatus(quizID uint, status string) error
}

// QuestionRepository - интерфейс для работы с вопросами
type QuestionRepository interface {
    Create(question *entity.Question) error
    GetByID(id uint) (*entity.Question, error)
    GetByQuizID(quizID uint) ([]entity.Question, error)
}

// ResultRepository - интерфейс для работы с результатами
type ResultRepository interface {
    SaveUserAnswer(answer *entity.UserAnswer) error
    GetUserAnswers(userID uint, quizID uint) ([]entity.UserAnswer, error)
    SaveResult(result *entity.Result) error
    GetQuizResults(quizID uint) ([]entity.Result, error)
    GetUserResult(userID uint, quizID uint) (*entity.Result, error)
}

// CacheRepository - интерфейс для работы с кешем
type CacheRepository interface {
    Set(key string, value interface{}, expiration time.Duration) error
    Get(key string) (string, error)
    Delete(key string) error
    Increment(key string) (int64, error)
}
```

## 4. План разработки бэкенда

### 4.1. Этап 1: Настройка проекта и базовая инфраструктура

1. **Настройка проекта**
   - Инициализация Go-модуля
   - Настройка структуры каталогов
   - Настройка конфигурации

2. **Настройка баз данных**
   - Настройка PostgreSQL для хранения данных
   - Настройка Redis для кеширования и WebSocket
   - Создание миграций для базы данных

3. **Аутентификация**
   - Реализация регистрации пользователей
   - Реализация JWT-аутентификации
   - Создание middleware для защиты маршрутов

### 4.2. Этап 2: Базовая бизнес-логика и API

1. **Реализация репозиториев**
   - Репозиторий пользователей
   - Репозиторий викторин
   - Репозиторий вопросов
   - Репозиторий результатов

2. **Реализация сервисов**
   - Сервис аутентификации
   - Сервис викторин
   - Сервис результатов

3. **Реализация REST API**
   - Регистрация и вход
   - Управление пользовательским профилем
   - Получение информации о викторине
   - Получение результатов

### 4.3. Этап 3: WebSocket и проведение викторины

1. **Реализация WebSocket инфраструктуры**
   - Настройка WebSocket Hub
   - Управление клиентскими соединениями
   - Маршрутизация сообщений

2. **Механизм проведения викторины**
   - Планирование викторины
   - Обратный отсчет и уведомления
   - Отправка вопросов
   - Обработка ответов
   - Подсчет результатов

3. **Кэширование и оптимизация**
   - Кэширование данных викторины
   - Оптимизация доступа к базе данных
   - Улучшение производительности WebSocket

### 4.4. Этап 4: Тестирование и безопасность

1. **Модульное тестирование**
   - Тестирование репозиториев
   - Тестирование сервисов
   - Тестирование WebSocket логики

2. **Интеграционное тестирование**
   - Тестирование API
   - Тестирование процесса викторины
   - Тестирование синхронизации

3. **Безопасность**
   - Защита от CSRF и XSS
   - Защита от DDoS
   - Валидация данных и защита от инъекций

## 5. Ключевые API-эндпоинты

### 5.1. REST API

```
// Аутентификация
POST   /api/auth/register      - Регистрация нового пользователя
POST   /api/auth/login         - Вход в систему
POST   /api/auth/logout        - Выход из системы

// Пользователи
GET    /api/users/me           - Получение информации о текущем пользователе
PUT    /api/users/me           - Обновление информации о пользователе
GET    /api/users/me/results   - Получение результатов пользователя

// Викторина
GET    /api/quiz/active        - Получение информации об активной викторине
GET    /api/quiz/:id           - Получение информации о конкретной викторине
GET    /api/quiz/:id/results   - Получение результатов викторины

// Административные (требуют прав админа)
POST   /api/admin/quiz         - Создание новой викторины
PUT    /api/admin/quiz/:id     - Обновление викторины
POST   /api/admin/quiz/:id/questions - Добавление вопросов
PUT    /api/admin/quiz/:id/schedule  - Планирование времени викторины
```

### 5.2. WebSocket

```
// Соединение с сервером WebSocket
GET    /ws                     - Подключение к WebSocket (с токеном авторизации)

// События от клиента к серверу
- user:ready                   - Пользователь готов к викторине
- user:answer                  - Ответ пользователя на вопрос
- user:heartbeat               - Проверка соединения

// События от сервера к клиенту
- quiz:announcement            - Анонс викторины
- quiz:countdown               - Обратный отсчет
- quiz:start                   - Начало викторины
- quiz:question                - Новый вопрос
- quiz:timer                   - Обновление таймера
- quiz:answer_result           - Результат ответа
- quiz:stats                   - Статистика после вопроса
- quiz:end                     - Конец викторины
- quiz:leaderboard             - Таблица лидеров
```

## 6. Схема взаимодействия компонентов

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│     HTTP Handler    │     │   WebSocket Handler │     │   Quiz Scheduler    │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
           │                          │                           │
           ▼                          ▼                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                    Service Layer                             │
├─────────────────────┬─────────────────────┬─────────────────────────────────┤
│    Auth Service     │    Quiz Service     │          Result Service         │
└─────────────────────┴─────────────────────┴─────────────────────────────────┘
           │                          │                           │
           ▼                          ▼                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                Repository Layer                              │
├─────────────────────┬─────────────────────┬─────────────────────────────────┤
│    User Repository  │   Quiz Repository   │        Result Repository        │
└─────────────────────┴─────────────────────┴─────────────────────────────────┘
           │                          │                           │
           ▼                          ▼                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                  Storage Layer                               │
├─────────────────────────────────┬─────────────────────────────────────────┬─┘
│         PostgreSQL              │              Redis                       │
└─────────────────────────────────┴─────────────────────────────────────────┘
```

## 7. Пример кода для ключевых компонентов

### 7.1 Основной файл WebSocket Hub

```go
// internal/websocket/hub.go
package websocket

import (
    "log"
    "sync"
)

// Hub поддерживает набор активных клиентов и транслирует сообщения
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
}

// NewHub создает новый экземпляр Hub
func NewHub() *Hub {
    return &Hub{
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
        userMap:    make(map[string]*Client),
    }
}

// Run запускает цикл обработки сообщений Hub
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            if client.UserID != "" {
                h.userMap[client.UserID] = client
            }
            h.mu.Unlock()
            
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                if client.UserID != "" {
                    delete(h.userMap, client.UserID)
                }
                close(client.send)
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
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

// BroadcastJSON отправляет структуру JSON всем клиентам
func (h *Hub) BroadcastJSON(v interface{}) error {
    // Реализация JSON-сериализации и отправки
    // ...
}

// SendToUser отправляет сообщение конкретному пользователю
func (h *Hub) SendToUser(userID string, message []byte) bool {
    h.mu.RLock()
    client, exists := h.userMap[userID]
    h.mu.RUnlock()
    
    if exists {
        select {
        case client.send <- message:
            return true
        default:
            h.mu.Lock()
            delete(h.clients, client)
            delete(h.userMap, userID)
            close(client.send)
            h.mu.Unlock()
            return false
        }
    }
    return false
}
```

### 7.2 Менеджер проведения викторины

```go
// internal/service/quiz_manager.go
package service

import (
    "context"
    "log"
    "sync"
    "time"
    
    "trivia-api/internal/domain/entity"
    "trivia-api/internal/domain/repository"
    "trivia-api/internal/websocket"
)

// QuizManager управляет процессом викторины
type QuizManager struct {
    quizRepo    repository.QuizRepository
    questionRepo repository.QuestionRepository
    resultRepo  repository.ResultRepository
    cacheRepo   repository.CacheRepository
    wsHub       *websocket.Hub
    
    // Текущая активная викторина
    activeQuiz    *entity.Quiz
    activeQuizMu  sync.RWMutex
    
    // Канал для остановки активных таймеров
    stopChan    chan struct{}
}

// NewQuizManager создает новый экземпляр менеджера викторины
func NewQuizManager(
    quizRepo repository.QuizRepository,
    questionRepo repository.QuestionRepository,
    resultRepo repository.ResultRepository,
    cacheRepo repository.CacheRepository,
    wsHub *websocket.Hub,
) *QuizManager {
    return &QuizManager{
        quizRepo:    quizRepo,
        questionRepo: questionRepo,
        resultRepo:  resultRepo,
        cacheRepo:   cacheRepo,
        wsHub:       wsHub,
        stopChan:    make(chan struct{}),
    }
}

// ScheduleQuiz планирует запуск викторины в заданное время
func (qm *QuizManager) ScheduleQuiz(quizID uint, scheduledTime time.Time) error {
    // Получаем викторину
    quiz, err := qm.quizRepo.GetWithQuestions(quizID)
    if err != nil {
        return err
    }
    
    // Устанавливаем время запуска
    quiz.ScheduledTime = scheduledTime
    quiz.Status = "scheduled"
    
    // Сохраняем изменения
    if err := qm.quizRepo.Update(quiz); err != nil {
        return err
    }
    
    // Вычисляем время до запуска
    timeToStart := scheduledTime.Sub(time.Now())
    if timeToStart <= 0 {
        return nil // Время уже прошло
    }
    
    // Запускаем горутину с таймером
    go func() {
        // Отправляем уведомление за 30 минут (если это возможно)
        notifyTime := scheduledTime.Add(-30 * time.Minute)
        if notifyTime.After(time.Now()) {
            timeToNotify := notifyTime.Sub(time.Now())
            select {
            case <-time.After(timeToNotify):
                qm.sendQuizAnnouncement(quiz)
            case <-qm.stopChan:
                return
            }
        }
        
        // Открываем комнату ожидания за 5 минут
        waitingRoomTime := scheduledTime.Add(-5 * time.Minute)
        if waitingRoomTime.After(time.Now()) {
            timeToWaitingRoom := waitingRoomTime.Sub(time.Now())
            select {
            case <-time.After(timeToWaitingRoom):
                qm.openWaitingRoom(quiz)
            case <-qm.stopChan:
                return
            }
        }
        
        // Запускаем викторину в назначенное время
        timeToStart := scheduledTime.Sub(time.Now())
        if timeToStart > 0 {
            select {
            case <-time.After(timeToStart):
                qm.startQuiz(quiz)
            case <-qm.stopChan:
                return
            }
        }
    }()
    
    return nil
}

// startQuiz запускает процесс викторины
func (qm *QuizManager) startQuiz(quiz *entity.Quiz) {
    // Устанавливаем викторину как активную
    qm.activeQuizMu.Lock()
    qm.activeQuiz = quiz
    qm.activeQuizMu.Unlock()
    
    // Обновляем статус в БД
    qm.quizRepo.UpdateStatus(quiz.ID, "in_progress")
    
    // Отправляем сообщение о начале викторины
    startEvent := websocket.Event{
        Type: "quiz:start",
        Data: map[string]interface{}{
            "quiz_id": quiz.ID,
            "title": quiz.Title,
            "question_count": len(quiz.Questions),
        },
    }
    qm.wsHub.BroadcastJSON(startEvent)
    
    // Начинаем последовательно отправлять вопросы
    qm.runQuizQuestions(quiz)
}

// runQuizQuestions последовательно отправляет вопросы и управляет таймерами
func (qm *QuizManager) runQuizQuestions(quiz *entity.Quiz) {
    for i, question := range quiz.Questions {
        // Отправляем вопрос всем участникам
        questionEvent := websocket.Event{
            Type: "quiz:question",
            Data: map[string]interface{}{
                "question_id": question.ID,
                "quiz_id": quiz.ID,
                "number": i + 1,
                "text": question.Text,
                "options": question.Options,
                "time_limit": question.TimeLimitSec,
                "total_questions": len(quiz.Questions),
            },
        }
        qm.wsHub.BroadcastJSON(questionEvent)
        
        // Сохраняем время начала вопроса для подсчета времени ответа
        questionStartKey := fmt.Sprintf("question:%d:start_time", question.ID)
        qm.cacheRepo.Set(questionStartKey, time.Now().UnixNano()/int64(time.Millisecond), time.Hour)
        
        // Запускаем таймер для вопроса
        timeLimit := time.Duration(question.TimeLimitSec) * time.Second
        
        // Отправляем обновления таймера каждую секунду
        ticker := time.NewTicker(1 * time.Second)
        go func(q entity.Question, endTime time.Time) {
            defer ticker.Stop()
            
            for {
                select {
                case <-ticker.C:
                    remaining := int(endTime.Sub(time.Now()).Seconds())
                    if remaining <= 0 {
                        return
                    }
                    
                    // Отправляем обновление таймера
                    timerEvent := websocket.Event{
                        Type: "quiz:timer",
                        Data: map[string]interface{}{
                            "question_id": q.ID,
                            "remaining_seconds": remaining,
                        },
                    }
                    qm.wsHub.BroadcastJSON(timerEvent)
                case <-qm.stopChan:
                    return
                }
            }
        }(question, time.Now().Add(timeLimit))
        
        // Ждем завершения времени на вопрос
        time.Sleep(timeLimit)
        
        // Отправляем правильный ответ всем участникам
        answerRevealEvent := websocket.Event{
            Type: "quiz:answer_reveal",
            Data: map[string]interface{}{
                "question_id": question.ID,
                "correct_option": question.CorrectOption,
            },
        }
        qm.wsHub.BroadcastJSON(answerRevealEvent)
        
        // Пауза между вопросами
        if i < len(quiz.Questions)-1 {
            time.Sleep(5 * time.Second)
        }
    }
    
    // Завершаем викторину
    qm.finishQuiz(quiz.ID)
}

// finishQuiz завершает викторину и подсчитывает результаты
func (qm *QuizManager) finishQuiz(quizID uint) {
    // Обновляем статус в БД
    qm.quizRepo.UpdateStatus(quizID, "completed")
    
    // Очищаем активную викторину
    qm.activeQuizMu.Lock()
    qm.activeQuiz = nil
    qm.activeQuizMu.Unlock()
    
    // Получаем и подсчитываем результаты
    results, err := qm.calculateResults(quizID)
    if err != nil {
        log.Printf("Ошибка подсчета результатов: %v", err)
        return
    }
    
    // Отправляем сообщение о завершении викторины
    endEvent := websocket.Event{
        Type: "quiz:end",
        Data: map[string]interface{}{
            "quiz_id": quizID,
            "message": "Викторина завершена",
        },
    }
    qm.wsHub.BroadcastJSON(endEvent)
    
    // Отправляем результаты
    leaderboardEvent := websocket.Event{
        Type: "quiz:leaderboard",
        Data: map[string]interface{}{
            "quiz_id": quizID,
            "results": results,
        },
    }
    qm.wsHub.BroadcastJSON(leaderboardEvent)
}
```

## 8. Предварительный график разработки

| Этап | Описание | Продолжительность |
|------|----------|------------------|
| 1 | Настройка проекта и инфраструктуры | 1-2 недели |
| 2 | Базовая бизнес-логика и API | 2-3 недели |
| 3 | WebSocket и проведение викторины | 2-3 недели |
| 4 | Тестирование и безопасность | 1-2 недели |
| 5 | Деплой и финальные настройки | 1 неделя |

**Общая оценка времени**: 7-11 недель в зависимости от сложности реализации и количества разработчиков
