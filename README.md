# Trivia API - Бэкенд для приложения викторины

Это серверная часть приложения для проведения викторин в реальном времени. Сервер реализован на Golang с использованием Gin, GORM, Redis и WebSocket.

## Особенности

- RESTful API для управления викторинами и пользователями
- WebSocket для обмена сообщениями в реальном времени
- Синхронизированная игровая логика
- Аутентификация на основе JWT
- Хранение данных в PostgreSQL
- Кеширование с использованием Redis
- Контейнеризация с Docker

## Требования

- Go 1.20 или новее
- PostgreSQL 13 или новее
- Redis 6 или новее
- Docker и Docker Compose (для запуска в контейнерах)

## Структура проекта

```
/trivia-api
├── cmd/                      # Точки входа приложения
│   └── api/                  # Основное API приложение
│       └── main.go           # Основной файл запуска
│
├── config/                   # Конфигурация приложения
│   └── config.yaml           # Файл конфигурации
│
├── internal/                 # Внутренний код приложения
│   ├── config/               # Конфигурация приложения
│   ├── domain/               # Бизнес-сущности и интерфейсы
│   ├── service/              # Бизнес-логика
│   ├── handler/              # Обработчики запросов
│   ├── middleware/           # Промежуточное ПО
│   ├── repository/           # Реализации репозиториев
│   └── websocket/            # WebSocket логика
│
├── pkg/                      # Переиспользуемые пакеты
│   ├── auth/                 # Аутентификация
│   ├── database/             # Работа с БД
│   └── logger/               # Логирование
│
├── migrations/               # Миграции БД
│
├── Dockerfile                # Dockerfile для сборки образа
├── docker-compose.yml        # Конфигурация Docker Compose
├── Makefile                  # Makefile с полезными командами
└── README.md                # Документация проекта
```

## Быстрый старт

### Запуск с использованием Docker

```bash
# Клонировать репозиторий
git clone https://github.com/yourusername/trivia-api.git
cd trivia-api

# Запустить приложение с использованием Docker Compose
make docker-up

# Остановить приложение
make docker-down
```

### Запуск для разработки

```bash
# Клонировать репозиторий
git clone https://github.com/yourusername/trivia-api.git
cd trivia-api

# Инициализировать проект
make init

# Запустить PostgreSQL и Redis (например, через Docker)
docker-compose up -d postgres redis

# Применить миграции
make migrate-up

# Запустить сервер в режиме разработки
make run
```

## API эндпоинты

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
```

### Административное API

```
POST   /api/quizzes                  - Создание новой викторины
POST   /api/quizzes/:id/questions    - Добавление вопросов к викторине
PUT    /api/quizzes/:id/schedule     - Планирование времени викторины
PUT    /api/quizzes/:id/cancel       - Отмена викторины
```

### WebSocket

```
GET    /ws                     - Подключение к WebSocket (с токеном авторизации)
```

## WebSocket-сообщения

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
server:heartbeat  - Ответ на проверку соединения
```

## Лицензия

Этот проект распространяется под лицензией MIT.