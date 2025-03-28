# Trivia Quiz Backend API

## Обзор проекта

Trivia Quiz Backend API - это серверное приложение для проведения викторин в реальном времени. Приложение обеспечивает одновременное участие множества пользователей, синхронизацию вопросов, таймеры для ответов и подсчет результатов. В проекте реализованы REST API и WebSocket для обеспечения интерактивности в реальном времени.

## Технологический стек

- **Язык программирования**: Go 1.20+
- **Веб-фреймворк**: Gin
- **База данных**: PostgreSQL
- **Кеширование**: Redis
- **Аутентификация**: JWT
- **Реальное время**: WebSocket
- **Контейнеризация**: Docker, Docker Compose
- **Миграции БД**: golang-migrate

## Особенности

- **Аутентификация пользователей**: Регистрация, вход с использованием JWT
- **Администрирование викторин**: Создание, планирование, управление вопросами
- **Синхронизированная игровая логика**: Все участники получают вопросы одновременно
- **Таймеры ответов**: Ограниченное время на каждый вопрос
- **Подсчет результатов в реальном времени**: Мгновенная обратная связь после каждого ответа
- **Рейтинговая система**: Подсчет очков с учетом скорости ответа
- **Защищенные API**: Разграничение доступа для обычных пользователей и администраторов
- **Оптимизированное хранение данных**: Эффективная структура БД с индексацией

## Структура проекта

```
/trivia-api
├── cmd/                      # Точки входа приложения
│   ├── api/                  # Основное API приложение
│   │   └── main.go           # Основной файл запуска
│   └── tools/                # Вспомогательные инструменты
│       └── create_admin.go   # Создание администратора
│
├── config/                   # Конфигурация приложения
│   └── config.yaml           # Файл конфигурации
│
├── internal/                 # Внутренний код приложения
│   ├── config/               # Загрузка конфигурации
│   ├── domain/               # Бизнес-сущности и интерфейсы
│   │   ├── entity/           # Модели данных
│   │   └── repository/       # Интерфейсы репозиториев
│   ├── service/              # Бизнес-логика
│   ├── handler/              # Обработчики запросов
│   ├── middleware/           # Промежуточное ПО
│   ├── repository/           # Реализации репозиториев
│   │   ├── postgres/         # PostgreSQL репозитории
│   │   └── redis/            # Redis репозиторий
│   └── websocket/            # WebSocket логика
│
├── pkg/                      # Переиспользуемые пакеты
│   ├── auth/                 # JWT аутентификация
│   └── database/             # Подключение к БД
│
├── migrations/               # SQL миграции
│
├── Dockerfile                # Dockerfile для сборки
├── docker-compose.yml        # Конфигурация Docker Compose
├── Makefile                  # Команды для сборки и запуска
└── README.md                 # Документация проекта
```

## Бизнес-процесс викторины

1. **Подготовка к викторине**:
   - Администратор создает викторину и добавляет вопросы
   - Администратор устанавливает время начала викторины
   - Пользователи видят анонс и могут подготовиться

2. **Проведение викторины**:
   - За 30 минут до начала отправляется уведомление всем пользователям
   - За 5 минут открывается "зал ожидания"
   - За 1 минуту начинается обратный отсчет
   - В назначенное время автоматически стартует викторина
   - Все участники получают вопросы одновременно
   - Для каждого вопроса запускается таймер
   - После окончания времени или получения всех ответов показывается правильный ответ

3. **Результаты викторины**:
   - Подсчитывается количество правильных ответов
   - Учитывается скорость ответа (быстрее = больше очков)
   - Формируется рейтинг участников
   - Отображаются итоговые результаты

## Установка и запуск

### Предварительные требования

- Go 1.20 или новее
- PostgreSQL 13 или новее
- Redis 6 или новее
- Docker и Docker Compose (опционально)

### Клонирование репозитория

```bash
git clone https://github.com/yourusername/trivia-api.git
cd trivia-api
```

### Установка зависимостей

```bash
go mod download
```

### Настройка конфигурации

Отредактируйте файл `config/config.yaml` в соответствии с вашей средой:

```yaml
server:
  port: "8080"
  readTimeout: 10
  writeTimeout: 10

database:
  host: "localhost"
  port: "5432"
  user: "postgres"
  password: "your_password"
  dbname: "trivia_db"
  sslmode: "disable"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

jwt:
  secret: "your_super_secret_key_change_in_production"
  expirationHrs: 24
```

### Создание базы данных

```bash
# Подключение к PostgreSQL
psql -U postgres

# В консоли PostgreSQL
CREATE DATABASE trivia_db;
\q
```

### Применение миграций

```bash
# Установка инструмента для миграций
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Применение миграций
migrate -path migrations -database "postgres://postgres:your_password@localhost:5432/trivia_db?sslmode=disable" up
```

### Запуск приложения

#### Локальная разработка

```bash
# Сборка приложения
go build -o trivia-api ./cmd/api

# Запуск приложения
./trivia-api
```

#### Использование Docker

```bash
# Сборка и запуск с Docker Compose
docker-compose up -d
```

### Создание администратора

```bash
go run ./cmd/tools/create_admin.go
```

## API Endpoints

### Аутентификация

```
POST   /api/auth/register      - Регистрация нового пользователя
POST   /api/auth/login         - Вход в систему
```

### Пользователи

```
GET    /api/users/me           - Получение информации о текущем пользователе
PUT    /api/users/me           - Обновление информации о пользователе
```

### Викторины

```
GET    /api/quizzes                  - Список всех викторин
GET    /api/quizzes/active           - Получение активной викторины
GET    /api/quizzes/scheduled        - Получение запланированных викторин
GET    /api/quizzes/:id              - Получение информации о викторине
GET    /api/quizzes/:id/with-questions - Получение викторины с вопросами
GET    /api/quizzes/:id/results      - Получение результатов викторины
GET    /api/quizzes/:id/my-result    - Получение результата текущего пользователя

# Маршруты для администраторов
POST   /api/quizzes                  - Создание новой викторины
POST   /api/quizzes/:id/questions    - Добавление вопросов к викторине
PUT    /api/quizzes/:id/schedule     - Планирование времени викторины
PUT    /api/quizzes/:id/cancel       - Отмена викторины
```

### WebSocket

```
GET    /ws                     - Подключение к WebSocket (с токеном авторизации)
```

## WebSocket события

### От клиента к серверу

```
user:ready      - Пользователь готов к викторине
user:answer     - Ответ пользователя на вопрос
user:heartbeat  - Проверка соединения
```

### От сервера к клиенту

```
quiz:announcement - Анонс викторины
quiz:waiting_room - Открытие зала ожидания
quiz:countdown    - Обратный отсчет
quiz:start        - Начало викторины
quiz:question     - Новый вопрос
quiz:timer        - Обновление таймера
quiz:answer_reveal - Показ правильного ответа
quiz:answer_result - Результат ответа
quiz:end          - Конец викторины
quiz:leaderboard  - Таблица лидеров
quiz:user_ready   - Уведомление о готовности пользователя
quiz:cancelled    - Уведомление об отмене викторины
server:heartbeat  - Ответ на проверку соединения
error             - Сообщение об ошибке
```

## Примеры использования

### Регистрация пользователя

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Вход в систему

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Создание викторины (требуются права администратора)

```bash
curl -X POST http://localhost:8080/api/quizzes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "Общие знания",
    "description": "Викторина по общим знаниям",
    "scheduled_time": "2025-04-01T18:00:00Z"
  }'
```

### Добавление вопросов (требуются права администратора)

```bash
curl -X POST http://localhost:8080/api/quizzes/1/questions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "questions": [
      {
        "text": "Какая планета известна как Красная планета?",
        "options": ["Венера", "Марс", "Юпитер", "Сатурн"],
        "correct_option": 1,
        "time_limit_sec": 15,
        "point_value": 10
      },
      {
        "text": "Кто написал 'Война и мир'?",
        "options": ["Достоевский", "Толстой", "Чехов", "Пушкин"],
        "correct_option": 1,
        "time_limit_sec": 15,
        "point_value": 10
      }
    ]
  }'
```

### Подключение к WebSocket

```javascript
// В JavaScript клиенте
const ws = new WebSocket('ws://localhost:8080/ws?token=YOUR_JWT_TOKEN');

// Обработка событий
ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'quiz:question':
      // Отобразить вопрос
      break;
    case 'quiz:timer':
      // Обновить таймер
      break;
    // Другие обработчики...
  }
};

// Отправка ответа
ws.send(JSON.stringify({
  type: 'user:answer',
  data: {
    question_id: 123,
    selected_option: 2,
    timestamp: Date.now()
  }
}));
```

## Разработка

### Модульное тестирование

```bash
go test ./...
```

### Сборка для различных платформ

```bash
# Для Windows
GOOS=windows GOARCH=amd64 go build -o trivia-api.exe ./cmd/api

# Для macOS
GOOS=darwin GOARCH=amd64 go build -o trivia-api-mac ./cmd/api

# Для Linux
GOOS=linux GOARCH=amd64 go build -o trivia-api-linux ./cmd/api
```

## Дополнительные команды Makefile

```bash
# Запуск в режиме разработки
make run

# Сборка приложения
make build

# Запуск Docker контейнеров
make docker-up

# Остановка Docker контейнеров
make docker-down

# Применение миграций
make migrate-up

# Откат миграций
make migrate-down

# Создание администратора
make create-admin
```

## Примечания по безопасности

- Пароли хешируются с использованием bcrypt
- API защищены с помощью JWT
- Административные функции доступны только администраторам
- WebSocket соединения требуют аутентификации

## Лицензия

Этот проект распространяется под лицензией MIT.

## Авторы

Ваше имя и контактная информация

## Вклад в проект

Инструкции по вкладу в проект, правила форматирования кода, процесс создания Pull Request и т.д.
