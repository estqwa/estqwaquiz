# Схема базы данных

## Обзор

База данных Trivia API предназначена для хранения информации о пользователях, викторинах, вопросах, ответах и результатах. Система использует PostgreSQL в качестве основной СУБД для хранения всех постоянных данных.

## Основные таблицы

### Пользователи (users)

Таблица `users` хранит информацию о зарегистрированных пользователях системы.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор пользователя |
| username | VARCHAR(50) | Имя пользователя (уникальное) |
| email | VARCHAR(255) | Email пользователя (уникальный) |
| password_hash | VARCHAR(255) | Хеш пароля пользователя |
| role | VARCHAR(20) | Роль пользователя (user, admin, moderator) |
| created_at | TIMESTAMP | Дата и время создания учетной записи |
| updated_at | TIMESTAMP | Дата и время последнего обновления |
| avatar_url | VARCHAR(255) | URL аватара пользователя |
| is_active | BOOLEAN | Статус активности учетной записи |
| settings | JSONB | Пользовательские настройки в формате JSON |

Индексы:
- users_username_idx (username)
- users_email_idx (email)
- users_role_idx (role)

### Викторины (quizzes)

Таблица `quizzes` содержит информацию о созданных викторинах.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор викторины |
| title | VARCHAR(255) | Название викторины |
| description | TEXT | Описание викторины |
| category | VARCHAR(100) | Категория викторины |
| difficulty | VARCHAR(20) | Сложность (easy, medium, hard) |
| creator_id | INTEGER REFERENCES users(id) | Создатель викторины |
| is_public | BOOLEAN | Флаг публичной доступности |
| start_time | TIMESTAMP | Запланированное время начала |
| end_time | TIMESTAMP | Запланированное время окончания |
| duration_minutes | INTEGER | Продолжительность в минутах |
| question_count | INTEGER | Количество вопросов |
| created_at | TIMESTAMP | Дата и время создания |
| updated_at | TIMESTAMP | Дата и время обновления |
| settings | JSONB | Настройки викторины в формате JSON |
| status | VARCHAR(20) | Статус викторины (draft, published, active, completed) |

Индексы:
- quizzes_creator_id_idx (creator_id)
- quizzes_category_idx (category)
- quizzes_status_idx (status)
- quizzes_start_time_idx (start_time)

### Вопросы (questions)

Таблица `questions` содержит вопросы, которые используются в викторинах.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор вопроса |
| quiz_id | INTEGER REFERENCES quizzes(id) | Идентификатор викторины |
| text | TEXT | Текст вопроса |
| type | VARCHAR(30) | Тип вопроса (single_choice, multiple_choice, text) |
| points | INTEGER | Количество баллов за правильный ответ |
| time_limit_seconds | INTEGER | Ограничение времени ответа в секундах |
| media_url | VARCHAR(255) | URL медиа-ресурса к вопросу |
| options | JSONB | Варианты ответов в формате JSON |
| correct_answers | JSONB | Правильные ответы в формате JSON |
| hint | TEXT | Подсказка к вопросу |
| explanation | TEXT | Объяснение правильного ответа |
| order_num | INTEGER | Порядковый номер вопроса в викторине |
| created_at | TIMESTAMP | Дата и время создания |
| updated_at | TIMESTAMP | Дата и время обновления |

Индексы:
- questions_quiz_id_idx (quiz_id)
- questions_type_idx (type)
- questions_order_num_idx (quiz_id, order_num)

### Ответы пользователей (user_answers)

Таблица `user_answers` хранит ответы пользователей на вопросы викторины.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор ответа |
| user_id | INTEGER REFERENCES users(id) | Идентификатор пользователя |
| quiz_id | INTEGER REFERENCES quizzes(id) | Идентификатор викторины |
| question_id | INTEGER REFERENCES questions(id) | Идентификатор вопроса |
| answer_data | JSONB | Данные ответа пользователя в формате JSON |
| is_correct | BOOLEAN | Флаг правильности ответа |
| points_earned | INTEGER | Заработанные баллы |
| answer_time_ms | INTEGER | Время ответа в миллисекундах |
| submitted_at | TIMESTAMP | Дата и время отправки ответа |
| client_info | JSONB | Информация о клиенте в формате JSON |

Индексы:
- user_answers_user_id_idx (user_id)
- user_answers_quiz_id_idx (quiz_id)
- user_answers_question_id_idx (question_id)
- user_answers_composite_idx (user_id, quiz_id, question_id)

### Результаты (results)

Таблица `results` содержит итоговые результаты участия пользователей в викторинах.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор результата |
| user_id | INTEGER REFERENCES users(id) | Идентификатор пользователя |
| quiz_id | INTEGER REFERENCES quizzes(id) | Идентификатор викторины |
| total_points | INTEGER | Общее количество баллов |
| correct_answers | INTEGER | Количество правильных ответов |
| total_questions | INTEGER | Общее количество вопросов в викторине |
| completion_time_ms | INTEGER | Время прохождения в миллисекундах |
| rank | INTEGER | Место в рейтинге |
| completed_at | TIMESTAMP | Дата и время завершения |
| detailed_results | JSONB | Детальная статистика в формате JSON |

Индексы:
- results_user_id_idx (user_id)
- results_quiz_id_idx (quiz_id)
- results_rank_idx (quiz_id, rank)
- results_user_quiz_idx (user_id, quiz_id)

### Недействительные токены (invalid_tokens)

Таблица `invalid_tokens` хранит информацию о токенах JWT, которые были отозваны до истечения их срока действия.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор записи |
| token_id | VARCHAR(255) | Идентификатор токена (jti) |
| expiration | TIMESTAMP | Время истечения токена |
| invalidated_at | TIMESTAMP | Время признания токена недействительным |
| reason | VARCHAR(100) | Причина признания токена недействительным |

Индексы:
- invalid_tokens_token_id_idx (token_id)
- invalid_tokens_expiration_idx (expiration)

### Обновляемые токены (refresh_tokens)

Таблица `refresh_tokens` предназначена для хранения refresh-токенов, используемых для обновления JWT без повторной аутентификации.

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL PRIMARY KEY | Уникальный идентификатор записи |
| user_id | INTEGER REFERENCES users(id) | Идентификатор пользователя |
| token_hash | VARCHAR(255) | Хеш токена обновления |
| expires_at | TIMESTAMP | Время истечения токена |
| issued_at | TIMESTAMP | Время выдачи токена |
| is_revoked | BOOLEAN | Флаг отозванности токена |
| device_info | VARCHAR(255) | Информация об устройстве |
| ip_address | VARCHAR(45) | IP-адрес, с которого был выпущен токен |
| previous_token_id | INTEGER | Ссылка на предыдущий токен в цепочке обновления |

Индексы:
- refresh_tokens_user_id_idx (user_id)
- refresh_tokens_token_hash_idx (token_hash)
- refresh_tokens_expires_idx (expires_at)

## Схема отношений

```
                  ┌─────────────┐
                  │    users    │
                  └──────┬──────┘
                         │
             ┌───────────┼───────────┐
             │           │           │
     ┌───────▼─────┐     │     ┌─────▼─────────┐
     │   quizzes   │     │     │ refresh_tokens│
     └───────┬─────┘     │     └───────────────┘
             │           │
             │           │
     ┌───────▼─────┐     │
     │  questions  │     │
     └───────┬─────┘     │
             │           │
             │           │
     ┌───────▼─────┐     │
     │user_answers ◄─────┘
     └───────┬─────┘
             │
             │
     ┌───────▼─────┐
     │   results   │
     └─────────────┘
```

## Миграции базы данных

Система использует SQL-миграции для управления схемой базы данных. Файлы миграций находятся в директории `/migrations`.

### Список миграций

| Номер | Имя файла | Описание |
|-------|-----------|----------|
| 000001 | init_schema | Создание начальной схемы с таблицами users, quizzes, questions, user_answers, results и invalid_tokens |
| 000002 | add_refresh_tokens | Добавление таблицы refresh_tokens для управления токенами обновления |
| 000003 | modify_refresh_tokens | Модификация таблицы refresh_tokens с добавлением дополнительных полей для безопасности |

### Запуск миграций

Для применения миграций используется утилита `migrate`:

```bash
migrate -path migrations -database "postgresql://username:password@localhost:5432/database?sslmode=disable" up
```

Для отката миграций:

```bash
migrate -path migrations -database "postgresql://username:password@localhost:5432/database?sslmode=disable" down
```

### Контроль целостности данных

База данных использует следующие механизмы для обеспечения целостности данных:

1. **Внешние ключи** - для поддержания ссылочной целостности между таблицами
2. **Уникальные индексы** - для предотвращения дублирования данных
3. **Проверочные ограничения** - для валидации значений полей
4. **Каскадное удаление** - для автоматического удаления связанных записей

## Оптимизация производительности

Для обеспечения высокой производительности при работе с данными используются следующие техники:

1. **Индексирование** - создание индексов на часто используемых в запросах полях
2. **Денормализация** - хранение предрассчитанных значений для уменьшения количества соединений
3. **JSON/JSONB поля** - для хранения гибкой структуры данных без необходимости создания дополнительных таблиц
4. **Партиционирование** - для таблиц с большим количеством данных (например, user_answers)

## Рекомендации по работе с базой данных

1. **Использование транзакций** - оборачивать операции, которые должны выполняться атомарно, в транзакции
2. **Подготовленные запросы** - использовать параметризованные запросы для повышения безопасности и производительности
3. **Ограничение размера запросов** - использовать пагинацию при получении больших наборов данных
4. **Регулярная очистка** - удалять устаревшие данные (например, недействительные токены с истекшим сроком)
5. **Мониторинг производительности** - отслеживать медленные запросы и оптимизировать их

## Настройка подключения

Параметры подключения к базе данных настраиваются через переменные окружения:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=trivia_api
DB_SSLMODE=disable
DB_MAX_CONNECTIONS=20
DB_MAX_IDLE_CONNECTIONS=10
```

Эти параметры используются в пакете `pkg/database` для установления соединения с PostgreSQL. 