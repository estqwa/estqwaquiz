# Документация по REST API

## Введение

REST API Trivia сервиса предоставляет ресурсы для создания и управления викторинами, учетными записями пользователей, вопросами и ответами. Это руководство описывает доступные эндпоинты, форматы запросов и ответов, а также включает примеры использования API.

## Базовая информация

### Базовый URL

Все запросы должны отправляться на базовый URL:

```
https://api.triviaserver.com/api/
```

Для локальной разработки:

```
http://localhost:8080/api/
```

### Формат ответа

Все ответы возвращаются в формате JSON и имеют следующую структуру:

#### Успешный ответ:

```json
{
  "success": true,
  "data": {
    // Данные, зависящие от запроса
  },
  "meta": {
    // Метаданные, такие как пагинация
  }
}
```

#### Ответ с ошибкой:

```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Описание ошибки"
  }
}
```

### HTTP коды ответов

API использует стандартные HTTP коды ответов:

| Код  | Описание                                |
|------|-----------------------------------------|
| 200  | OK - Запрос выполнен успешно           |
| 201  | Created - Ресурс успешно создан        |
| 400  | Bad Request - Некорректный запрос      |
| 401  | Unauthorized - Требуется аутентификация|
| 403  | Forbidden - Доступ запрещен            |
| 404  | Not Found - Ресурс не найден           |
| 422  | Unprocessable Entity - Ошибка валидации|
| 429  | Too Many Requests - Превышен лимит     |
| 500  | Internal Server Error - Внутренняя ошибка|

## Аутентификация

API поддерживает два метода аутентификации:

### 1. Cookie-based Authentication (для веб-приложений)

Этот метод использует HTTP-only cookie для хранения access token, что делает его более безопасным для веб-приложений. Дополнительно требуется CSRF-токен в заголовках.

#### Пример входа:

```javascript
// Вход в систему
fetch('https://api.triviaserver.com/api/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password123'
  }),
  credentials: 'include' // Важно для сохранения cookie
})
.then(response => response.json())
.then(data => {
  if (data.success) {
    // Сохраняем CSRF токен для последующих запросов
    localStorage.setItem('csrf_token', data.data.csrf_token);
  } else {
    console.error('Ошибка входа:', data.error);
  }
});

// Последующие запросы
fetch('https://api.triviaserver.com/api/user/profile', {
  method: 'GET',
  headers: {
    'X-CSRF-Token': localStorage.getItem('csrf_token')
  },
  credentials: 'include'
})
.then(response => response.json())
.then(data => {
  console.log('Профиль пользователя:', data.data);
});
```

#### Ответ при успешном входе:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 123,
      "email": "user@example.com",
      "username": "username",
      "role": "user"
    },
    "csrf_token": "f3g56h7j8k9l0..."
  }
}
```

### 2. Bearer Token Authentication (для мобильных и SPA приложений)

Этот метод использует токен доступа в заголовке Authorization.

#### Пример входа:

```javascript
// Вход в систему
fetch('https://api.triviaserver.com/api/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password123',
    token_based: true // Указываем, что хотим получить токен
  })
})
.then(response => response.json())
.then(data => {
  if (data.success) {
    // Сохраняем токены
    localStorage.setItem('access_token', data.data.access_token);
    localStorage.setItem('refresh_token', data.data.refresh_token);
  } else {
    console.error('Ошибка входа:', data.error);
  }
});

// Последующие запросы
fetch('https://api.triviaserver.com/api/user/profile', {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Профиль пользователя:', data.data);
});
```

#### Ответ при успешном входе:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 123,
      "email": "user@example.com",
      "username": "username",
      "role": "user"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

### Управление токенами

#### Обновление токена:

```javascript
fetch('https://api.triviaserver.com/api/auth/refresh', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    refresh_token: localStorage.getItem('refresh_token')
  })
})
.then(response => response.json())
.then(data => {
  if (data.success) {
    localStorage.setItem('access_token', data.data.access_token);
    localStorage.setItem('refresh_token', data.data.refresh_token);
  } else {
    console.error('Ошибка обновления токена:', data.error);
    // Редирект на страницу входа при необходимости
  }
});
```

#### Выход из системы:

```javascript
fetch('https://api.triviaserver.com/api/auth/logout', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  if (data.success) {
    // Очищаем локальное хранилище
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    // Редирект на страницу входа
  }
});
```

### Обработка 401 ошибок

При получении ответа 401 (Unauthorized), клиент должен попытаться обновить токен:

```javascript
async function fetchWithTokenRefresh(url, options = {}) {
  let response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${localStorage.getItem('access_token')}`
    }
  });

  if (response.status === 401) {
    // Пытаемся обновить токен
    const refreshResponse = await fetch('https://api.triviaserver.com/api/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        refresh_token: localStorage.getItem('refresh_token')
      })
    });
    
    const refreshData = await refreshResponse.json();
    
    if (refreshData.success) {
      // Сохраняем новые токены
      localStorage.setItem('access_token', refreshData.data.access_token);
      localStorage.setItem('refresh_token', refreshData.data.refresh_token);
      
      // Повторяем оригинальный запрос с новым токеном
      return fetch(url, {
        ...options,
        headers: {
          ...options.headers,
          'Authorization': `Bearer ${refreshData.data.access_token}`
        }
      });
    } else {
      // Если не удалось обновить токен, выходим из системы
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      window.location.href = '/login';
      throw new Error('Сессия истекла. Необходимо войти заново.');
    }
  }
  
  return response;
}
```

## Пагинация

Для ресурсов, которые могут возвращать большие наборы данных, API поддерживает пагинацию:

```javascript
fetch('https://api.triviaserver.com/api/quizzes?page=2&per_page=20', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Список викторин:', data.data);
  console.log('Метаданные пагинации:', data.meta.pagination);
});
```

Ответ с пагинацией:

```json
{
  "success": true,
  "data": [
    // Массив с данными
  ],
  "meta": {
    "pagination": {
      "total": 126,
      "per_page": 20,
      "current_page": 2,
      "last_page": 7,
      "next_page_url": "https://api.triviaserver.com/api/quizzes?page=3",
      "prev_page_url": "https://api.triviaserver.com/api/quizzes?page=1"
    }
  }
}
```

## Фильтрация и сортировка

API поддерживает фильтрацию и сортировку для многих эндпоинтов:

### Фильтрация:

```
GET /api/quizzes?filter[category]=science&filter[difficulty]=hard
```

### Сортировка:

```
GET /api/quizzes?sort=created_at
GET /api/quizzes?sort=-created_at // В обратном порядке
```

Пример в JavaScript:

```javascript
fetch('https://api.triviaserver.com/api/quizzes?filter[category]=science&sort=-created_at', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Отфильтрованные и отсортированные викторины:', data.data);
});
```

## Эндпоинты API

### Аутентификация

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /auth/register | Регистрация нового пользователя |
| POST | /auth/login | Вход в систему |
| POST | /auth/refresh | Обновление токена доступа |
| POST | /auth/logout | Выход из системы |
| POST | /auth/password/reset | Запрос сброса пароля |
| POST | /auth/password/reset/confirm | Подтверждение сброса пароля |

### Пользователи

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /users/me | Получение профиля текущего пользователя |
| PUT | /users/me | Обновление профиля текущего пользователя |
| GET | /users/{id} | Получение публичного профиля пользователя |
| GET | /users/me/statistics | Получение статистики пользователя |
| GET | /users/me/quizzes | Получение викторин пользователя |
| GET | /users/me/results | Получение результатов пользователя |
| POST | /users/me/avatar | Загрузка аватара пользователя |

### Викторины

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /quizzes | Получение списка викторин |
| POST | /quizzes | Создание новой викторины |
| GET | /quizzes/{id} | Получение информации о викторине |
| PUT | /quizzes/{id} | Обновление викторины |
| DELETE | /quizzes/{id} | Удаление викторины |
| GET | /quizzes/categories | Получение списка категорий |
| GET | /quizzes/active | Получение активных викторин |
| GET | /quizzes/scheduled | Получение запланированных викторин |
| GET | /quizzes/{id}/questions | Получение вопросов викторины |
| POST | /quizzes/{id}/questions | Добавление вопроса к викторине |
| GET | /quizzes/{id}/results | Получение результатов викторины |
| POST | /quizzes/{id}/join | Присоединение к викторине |
| POST | /quizzes/{id}/leave | Выход из викторины |

### Вопросы

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /questions/{id} | Получение информации о вопросе |
| PUT | /questions/{id} | Обновление вопроса |
| DELETE | /questions/{id} | Удаление вопроса |
| GET | /questions/{id}/options | Получение вариантов ответа |
| POST | /questions/{id}/answer | Отправка ответа на вопрос |

### Результаты

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /results/{id} | Получение результата |
| GET | /results/quiz/{quiz_id} | Получение результатов викторины |
| GET | /results/user/{user_id} | Получение результатов пользователя |
| GET | /results/leaderboard | Получение общей таблицы лидеров |
| GET | /results/leaderboard/{quiz_id} | Получение таблицы лидеров для викторины |

## Детальное описание эндпоинтов

### Регистрация пользователя

**Запрос:**

```http
POST /api/auth/register
Content-Type: application/json

{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "secure_password",
  "password_confirmation": "secure_password"
}
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 42,
      "username": "john_doe",
      "email": "john@example.com",
      "created_at": "2023-07-15T14:30:00Z"
    }
  }
}
```

**Ответ (422 Unprocessable Entity):**

```json
{
  "success": false,
  "error": "Ошибка валидации",
  "error_code": "VALIDATION_ERROR",
  "error_type": "validation_error",
  "details": {
    "email": ["Email уже используется"],
    "password": ["Пароль должен содержать минимум 8 символов"]
  }
}
```

### Вход в систему

**Запрос:**

```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "secure_password",
  "token_based": true
}
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 42,
      "username": "john_doe",
      "email": "john@example.com",
      "role": "user",
      "created_at": "2023-07-15T14:30:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

**Ответ (401 Unauthorized):**

```json
{
  "success": false,
  "error": "Неверный email или пароль",
  "error_code": "INVALID_CREDENTIALS",
  "error_type": "authentication_error"
}
```

### Получение профиля пользователя

**Запрос:**

```http
GET /api/user/profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 42,
      "username": "john_doe",
      "email": "john@example.com",
      "role": "user",
      "avatar_url": "https://api.triviaserver.com/storage/avatars/john_doe.jpg",
      "quiz_count": 5,
      "participation_count": 20,
      "created_at": "2023-07-15T14:30:00Z",
      "updated_at": "2023-07-20T10:15:00Z"
    }
  }
}
```

### Получение списка викторин

**Запрос:**

```http
GET /api/quizzes?category=history&difficulty=medium&page=1&per_page=10
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "title": "История Древнего Рима",
      "description": "Викторина о Римской империи",
      "category": "history",
      "difficulty": "medium",
      "creator": {
        "id": 5,
        "username": "history_buff"
      },
      "is_public": true,
      "question_count": 15,
      "duration_minutes": 30,
      "scheduled_time": "2023-08-15T18:00:00Z",
      "status": "scheduled",
      "created_at": "2023-07-10T12:00:00Z"
    },
    // ... другие викторины
  ],
  "meta": {
    "pagination": {
      "total": 25,
      "count": 10,
      "per_page": 10,
      "current_page": 1,
      "total_pages": 3,
      "next_page": 2,
      "prev_page": null
    }
  }
}
```

### Создание викторины

**Запрос:**

```http
POST /api/quizzes
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "title": "История Древнего Египта",
  "description": "Увлекательная викторина о Древнем Египте",
  "category": "history",
  "difficulty": "medium",
  "is_public": true,
  "duration_minutes": 25,
  "scheduled_time": "2023-09-10T19:00:00Z"
}
```

**Ответ (201 Created):**

```json
{
  "success": true,
  "data": {
    "id": 42,
    "title": "История Древнего Египта",
    "description": "Увлекательная викторина о Древнем Египте",
    "category": "history",
    "difficulty": "medium",
    "creator": {
      "id": 42,
      "username": "john_doe"
    },
    "is_public": true,
    "question_count": 0,
    "duration_minutes": 25,
    "scheduled_time": "2023-09-10T19:00:00Z",
    "status": "draft",
    "created_at": "2023-07-25T15:30:00Z"
  }
}
```

### Добавление вопроса к викторине

**Запрос:**

```http
POST /api/quizzes/42/questions
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "text": "Какая река протекает через Египет?",
  "time_limit_sec": 30,
  "point_value": 10,
  "options": [
    {
      "text": "Нил",
      "is_correct": true
    },
    {
      "text": "Тигр",
      "is_correct": false
    },
    {
      "text": "Евфрат",
      "is_correct": false
    },
    {
      "text": "Инд",
      "is_correct": false
    }
  ]
}
```

**Ответ (201 Created):**

```json
{
  "success": true,
  "data": {
    "id": 150,
    "quiz_id": 42,
    "text": "Какая река протекает через Египет?",
    "time_limit_sec": 30,
    "point_value": 10,
    "options": [
      {
        "id": 601,
        "text": "Нил",
        "is_correct": true
      },
      {
        "id": 602,
        "text": "Тигр",
        "is_correct": false
      },
      {
        "id": 603,
        "text": "Евфрат",
        "is_correct": false
      },
      {
        "id": 604,
        "text": "Инд",
        "is_correct": false
      }
    ],
    "created_at": "2023-07-25T15:35:00Z"
  }
}
```

### Присоединение к викторине

**Запрос:**

```http
POST /api/quizzes/42/join
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": {
    "message": "Вы успешно присоединились к викторине",
    "quiz": {
      "id": 42,
      "title": "История Древнего Египта",
      "scheduled_time": "2023-09-10T19:00:00Z",
      "status": "scheduled",
      "participant_count": 15
    }
  }
}
```

### Отправка ответа на вопрос

**Запрос:**

```http
POST /api/questions/150/answer
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "selected_option": 601,
  "answer_time_ms": 5230
}
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": {
    "is_correct": true,
    "points_earned": 10,
    "correct_option": 601
  }
}
```

### Получение результатов пользователя

**Запрос:**

```http
GET /api/users/me/results?page=1&per_page=10
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "id": 256,
      "quiz": {
        "id": 42,
        "title": "История Древнего Египта",
        "category": "history",
        "difficulty": "medium"
      },
      "score": 85,
      "correct_answers": 17,
      "total_questions": 20,
      "rank": 3,
      "completion_time_ms": 1126000,
      "completed_at": "2023-09-10T19:45:12Z"
    },
    // ... другие результаты
  ],
  "meta": {
    "pagination": {
      "total": 20,
      "count": 10,
      "per_page": 10,
      "current_page": 1,
      "total_pages": 2,
      "next_page": 2,
      "prev_page": null
    }
  }
}
```

### Получение таблицы лидеров для викторины

**Запрос:**

```http
GET /api/results/leaderboard/42?page=1&per_page=10
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Ответ (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "rank": 1,
      "user": {
        "id": 15,
        "username": "quiz_master",
        "avatar_url": "https://api.triviaserver.com/storage/avatars/quiz_master.jpg"
      },
      "score": 98,
      "correct_answers": 20,
      "total_questions": 20,
      "completion_time_ms": 953000
    },
    {
      "rank": 2,
      "user": {
        "id": 28,
        "username": "history_lover",
        "avatar_url": "https://api.triviaserver.com/storage/avatars/history_lover.jpg"
      },
      "score": 90,
      "correct_answers": 18,
      "total_questions": 20,
      "completion_time_ms": 1015000
    },
    // ... другие результаты
  ],
  "meta": {
    "pagination": {
      "total": 45,
      "count": 10,
      "per_page": 10,
      "current_page": 1,
      "total_pages": 5,
      "next_page": 2,
      "prev_page": null
    },
    "quiz": {
      "id": 42,
      "title": "История Древнего Египта",
      "category": "history",
      "difficulty": "medium",
      "completed_at": "2023-09-10T19:45:12Z"
    }
  }
}
```

## Обработка ошибок

### Типы ошибок

API может возвращать следующие типы ошибок:

- `validation_error` — Ошибка валидации данных
- `authentication_error` — Ошибка аутентификации
- `authorization_error` — Ошибка авторизации
- `not_found` — Ресурс не найден
- `rate_limit_exceeded` — Превышен лимит запросов
- `server_error` — Внутренняя ошибка сервера

### Пример обработки ошибок

```javascript
async function fetchData(url) {
  try {
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });
    
    const data = await response.json();
    
    if (!data.success) {
      // Обработка бизнес-ошибок
      switch (data.error_type) {
        case 'authentication_error':
          // Обновить токен или перенаправить на страницу входа
          refreshToken();
          break;
        case 'validation_error':
          // Показать ошибки валидации
          showValidationErrors(data.details);
          break;
        default:
          // Показать общую ошибку
          showError(data.error);
      }
      return null;
    }
    
    return data.data;
  } catch (error) {
    console.error('Network error:', error);
    showError('Ошибка сети. Пожалуйста, проверьте подключение.');
    return null;
  }
}
```

## Ограничения API

### Лимиты запросов

API имеет следующие ограничения по количеству запросов:

- Публичные эндпоинты: 60 запросов в минуту
- Авторизованные эндпоинты: 300 запросов в минуту

При превышении лимита API вернет ответ с кодом `429 Too Many Requests` и заголовком `Retry-After`, указывающим время (в секундах), через которое можно повторить запрос.

### Размер загружаемых файлов

- Аватар пользователя: максимум 2 МБ
- Изображения для вопросов: максимум 5 МБ

## Версионирование API

API использует версионирование в URL-пути (`/api/v1/`). При выпуске новой версии API предыдущие версии продолжают поддерживаться в течение минимум 6 месяцев.

## Рекомендации по использованию API

1. **Кэширование** — Кэшируйте данные, которые не меняются часто (категории, завершенные викторины и т.д.)
2. **Обработка ошибок** — Всегда проверяйте поле `success` в ответе и корректно обрабатывайте ошибки
3. **Обновление токенов** — Реализуйте логику обновления токена доступа при получении ошибки `401 Unauthorized`
4. **Пагинация** — Не пытайтесь получить все данные сразу, используйте пагинацию
5. **Логирование** — Логируйте ошибки API для упрощения отладки

## Контактная информация

Если у вас возникли вопросы или проблемы с API, обратитесь в службу поддержки:

- Email: api-support@triviaserver.com
- Документация: https://docs.triviaserver.com
- Статус API: https://status.triviaserver.com

## API Endpoints

### Авторизация и пользователи

#### Регистрация

```
POST /api/auth/register
```

Создает нового пользователя.

##### Параметры запроса:

| Параметр  | Тип    | Обязательный | Описание                 |
|-----------|--------|--------------|--------------------------|
| username  | string | Да           | Имя пользователя         |
| email     | string | Да           | Электронная почта        |
| password  | string | Да           | Пароль (мин. 8 символов) |
| full_name | string | Нет          | Полное имя               |

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/auth/register', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    username: 'newuser',
    email: 'user@example.com',
    password: 'password123',
    full_name: 'Иван Иванов'
  })
})
.then(response => response.json())
.then(data => {
  console.log('Регистрация успешна:', data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 456,
      "username": "newuser",
      "email": "user@example.com",
      "full_name": "Иван Иванов",
      "created_at": "2023-07-15T10:30:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

#### Профиль пользователя

```
GET /api/user/profile
```

Возвращает информацию о текущем пользователе.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/user/profile', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Профиль пользователя:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": {
    "id": 123,
    "username": "user123",
    "email": "user@example.com",
    "full_name": "Иван Иванов",
    "avatar_url": "https://api.triviaserver.com/images/avatars/123.jpg",
    "role": "user",
    "created_at": "2023-05-01T12:00:00Z",
    "stats": {
      "total_quizzes": 15,
      "quizzes_won": 3,
      "total_points": 1250,
      "average_score": 78.5
    }
  }
}
```

#### Обновление профиля

```
PUT /api/user/profile
```

Обновляет информацию профиля пользователя.

##### Параметры запроса:

| Параметр  | Тип    | Обязательный | Описание                 |
|-----------|--------|--------------|--------------------------|
| username  | string | Нет          | Новое имя пользователя   |
| full_name | string | Нет          | Новое полное имя         |
| avatar    | file   | Нет          | Новое изображение аватара|

##### Пример запроса:

```javascript
const formData = new FormData();
formData.append('username', 'newusername');
formData.append('full_name', 'Петр Петров');
formData.append('avatar', file); // Объект File из input[type=file]

fetch('https://api.triviaserver.com/api/user/profile', {
  method: 'PUT',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  },
  body: formData
})
.then(response => response.json())
.then(data => {
  console.log('Профиль обновлен:', data.data);
});
```

#### Изменение пароля

```
PUT /api/user/password
```

Обновляет пароль пользователя.

##### Параметры запроса:

| Параметр      | Тип    | Обязательный | Описание             |
|---------------|--------|--------------|----------------------|
| current_password | string | Да        | Текущий пароль       |
| new_password  | string | Да           | Новый пароль         |

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/user/password', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  },
  body: JSON.stringify({
    current_password: 'oldpassword123',
    new_password: 'newpassword456'
  })
})
.then(response => response.json())
.then(data => {
  console.log('Пароль изменен:', data.success);
});
```

#### Статистика пользователя

```
GET /api/user/stats
```

Возвращает детальную статистику пользователя.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/user/stats', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Статистика пользователя:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": {
    "total_quizzes": 42,
    "quizzes_completed": 38,
    "quizzes_won": 12,
    "total_points": 3750,
    "average_score": 82.5,
    "correct_answers": 320,
    "incorrect_answers": 80,
    "accuracy": 80.0,
    "fastest_answer_ms": 1250,
    "average_answer_time_ms": 4500,
    "categories": [
      {
        "name": "История",
        "quizzes_played": 10,
        "accuracy": 85.0
      },
      {
        "name": "Наука",
        "quizzes_played": 8,
        "accuracy": 78.5
      }
    ],
    "recent_achievements": [
      {
        "id": 5,
        "name": "Знаток истории",
        "description": "Правильно ответить на 50 вопросов по истории",
        "unlocked_at": "2023-06-20T15:30:00Z"
      }
    ]
  }
}
```

### Викторины (Quizzes)

#### Получение списка викторин

```
GET /api/quizzes
```

Возвращает список доступных викторин.

##### Параметры запроса:

| Параметр         | Тип    | Обязательный | Описание                     |
|------------------|--------|--------------|------------------------------|
| page             | int    | Нет          | Номер страницы (по умолчанию 1) |
| per_page         | int    | Нет          | Элементов на странице (по умолчанию 20) |
| filter[category] | string | Нет          | Фильтр по категории          |
| filter[status]   | string | Нет          | Фильтр по статусу: upcoming, active, completed |
| sort             | string | Нет          | Поле для сортировки (prefix - для обратной) |

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/quizzes?filter[category]=history&sort=-created_at&page=1&per_page=10', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Список викторин:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": [
    {
      "id": 123,
      "title": "История Древнего Рима",
      "description": "Викторина о Римской империи",
      "category": "История",
      "difficulty": "medium",
      "image_url": "https://api.triviaserver.com/images/quizzes/rome.jpg",
      "question_count": 20,
      "time_limit_sec": 30,
      "status": "upcoming",
      "start_time": "2023-08-15T18:00:00Z",
      "created_at": "2023-07-01T10:15:00Z",
      "created_by": {
        "id": 42,
        "username": "historyteacher"
      }
    },
    // ... другие викторины
  ],
  "meta": {
    "pagination": {
      "total": 35,
      "per_page": 10,
      "current_page": 1,
      "last_page": 4,
      "next_page_url": "https://api.triviaserver.com/api/quizzes?page=2",
      "prev_page_url": null
    }
  }
}
```

#### Получение отдельной викторины

```
GET /api/quizzes/{id}
```

Возвращает детальную информацию о викторине.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/quizzes/123', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Детали викторины:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": {
    "id": 123,
    "title": "История Древнего Рима",
    "description": "Викторина о Римской империи",
    "category": "История",
    "difficulty": "medium",
    "image_url": "https://api.triviaserver.com/images/quizzes/rome.jpg",
    "question_count": 20,
    "time_limit_sec": 30,
    "points_per_question": 10,
    "bonus_points": 5,
    "status": "upcoming",
    "start_time": "2023-08-15T18:00:00Z",
    "estimated_duration_min": 15,
    "created_at": "2023-07-01T10:15:00Z",
    "updated_at": "2023-07-05T14:30:00Z",
    "created_by": {
      "id": 42,
      "username": "historyteacher"
    },
    "participants_count": 156,
    "is_public": true,
    "tags": ["древний мир", "империи", "античность"]
  }
}
```

#### Создание викторины

```
POST /api/quizzes
```

Создает новую викторину.

##### Параметры запроса:

| Параметр          | Тип    | Обязательный | Описание                     |
|-------------------|--------|--------------|------------------------------|
| title             | string | Да           | Название викторины           |
| description       | string | Да           | Описание викторины           |
| category_id       | int    | Да           | ID категории                 |
| difficulty        | string | Да           | Сложность: easy, medium, hard|
| time_limit_sec    | int    | Да           | Время на вопрос (сек)        |
| start_time        | string | Нет          | Время начала (ISO 8601)      |
| image             | file   | Нет          | Изображение для викторины    |
| is_public         | boolean| Нет          | Публичная викторина (по умолчанию true) |
| tags              | array  | Нет          | Массив тегов                 |

##### Пример запроса:

```javascript
const formData = new FormData();
formData.append('title', 'История России');
formData.append('description', 'Викторина о ключевых моментах истории России');
formData.append('category_id', 3);
formData.append('difficulty', 'medium');
formData.append('time_limit_sec', 30);
formData.append('start_time', '2023-09-01T19:00:00Z');
formData.append('image', file); // Объект File из input[type=file]
formData.append('is_public', true);
formData.append('tags[0]', 'история');
formData.append('tags[1]', 'Россия');

fetch('https://api.triviaserver.com/api/quizzes', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  },
  body: formData
})
.then(response => response.json())
.then(data => {
  console.log('Викторина создана:', data.data);
});
```

#### Присоединение к викторине

```
POST /api/quizzes/{id}/join
```

Позволяет пользователю присоединиться к викторине.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/quizzes/123/join', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Присоединение к викторине:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": {
    "quiz_id": 123,
    "participant_id": 456,
    "join_code": "XYZ789", // Код для присоединения к приватной викторине
    "status": "joined",
    "joined_at": "2023-07-15T17:45:00Z",
    "waiting_room_url": "https://api.triviaserver.com/waiting-room/123"
  }
}
```

#### Получение результатов викторины

```
GET /api/quizzes/{id}/results
```

Возвращает результаты завершенной викторины.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/quizzes/123/results', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Результаты викторины:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": {
    "quiz": {
      "id": 123,
      "title": "История Древнего Рима",
      "total_participants": 150
    },
    "user_result": {
      "rank": 5,
      "score": 180,
      "correct_answers": 18,
      "incorrect_answers": 2,
      "average_answer_time_ms": 3500
    },
    "leaderboard": [
      {
        "rank": 1,
        "user": {
          "id": 42,
          "username": "history_expert",
          "avatar_url": "https://api.triviaserver.com/images/avatars/42.jpg"
        },
        "score": 198,
        "correct_answers": 20,
        "average_answer_time_ms": 2800
      },
      // ... другие участники
    ]
  }
}
```

### Вопросы и ответы

#### Получение списка вопросов для викторины (только для создателя)

```
GET /api/quizzes/{id}/questions
```

Возвращает список вопросов для конкретной викторины.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/quizzes/123/questions', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Вопросы викторины:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": [
    {
      "id": 456,
      "quiz_id": 123,
      "text": "В каком году был основан Рим?",
      "image_url": null,
      "time_limit_override_sec": null,
      "points": 10,
      "order": 1,
      "options": [
        {
          "id": 1001,
          "text": "753 до н.э.",
          "is_correct": true
        },
        {
          "id": 1002,
          "text": "527 до н.э.",
          "is_correct": false
        },
        {
          "id": 1003,
          "text": "1000 до н.э.",
          "is_correct": false
        },
        {
          "id": 1004,
          "text": "476 н.э.",
          "is_correct": false
        }
      ]
    },
    // ... другие вопросы
  ]
}
```

#### Добавление вопроса к викторине

```
POST /api/quizzes/{id}/questions
```

Добавляет новый вопрос к викторине.

##### Параметры запроса:

| Параметр             | Тип    | Обязательный | Описание                     |
|----------------------|--------|--------------|------------------------------|
| text                 | string | Да           | Текст вопроса                |
| options              | array  | Да           | Массив вариантов ответа      |
| options[].text       | string | Да           | Текст варианта ответа        |
| options[].is_correct | boolean| Да           | Является ли вариант правильным|
| image                | file   | Нет          | Изображение для вопроса      |
| time_limit_override_sec | int | Нет          | Переопределение времени для вопроса |
| points               | int    | Нет          | Количество очков за вопрос   |
| order                | int    | Нет          | Порядковый номер вопроса     |

##### Пример запроса:

```javascript
const formData = new FormData();
formData.append('text', 'Какой римский император построил Колизей?');
formData.append('options[0][text]', 'Юлий Цезарь');
formData.append('options[0][is_correct]', false);
formData.append('options[1][text]', 'Веспасиан');
formData.append('options[1][is_correct]', true);
formData.append('options[2][text]', 'Нерон');
formData.append('options[2][is_correct]', false);
formData.append('options[3][text]', 'Август');
formData.append('options[3][is_correct]', false);
formData.append('points', 15);
formData.append('order', 2);

fetch('https://api.triviaserver.com/api/quizzes/123/questions', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  },
  body: formData
})
.then(response => response.json())
.then(data => {
  console.log('Вопрос добавлен:', data.data);
});
```

#### Получение ответов пользователя

```
GET /api/quizzes/{quiz_id}/user-answers
```

Возвращает ответы пользователя на вопросы викторины.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/quizzes/123/user-answers', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Ответы пользователя:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": [
    {
      "question_id": 456,
      "question_text": "В каком году был основан Рим?",
      "selected_option_id": 1001,
      "selected_option_text": "753 до н.э.",
      "is_correct": true,
      "points_earned": 10,
      "answer_time_ms": 3250
    },
    // ... другие ответы
  ]
}
```

### Категории

#### Получение списка категорий

```
GET /api/categories
```

Возвращает список доступных категорий для викторин.

##### Пример запроса:

```javascript
fetch('https://api.triviaserver.com/api/categories', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
})
.then(response => response.json())
.then(data => {
  console.log('Категории:', data.data);
});
```

##### Пример ответа:

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "История",
      "description": "Исторические события и личности",
      "icon_url": "https://api.triviaserver.com/images/categories/history.png"
    },
    {
      "id": 2,
      "name": "Наука",
      "description": "Научные факты и открытия",
      "icon_url": "https://api.triviaserver.com/images/categories/science.png"
    },
    // ... другие категории
  ]
}
```

## Обработка ошибок

### Типичные ошибки

| Код ошибки               | Описание                                           |
|--------------------------|---------------------------------------------------|
| invalid_credentials      | Неверные учетные данные                           |
| token_expired            | Токен доступа истек                               |
| token_invalid            | Недействительный токен                            |
| permission_denied        | Отказано в доступе                                |
| resource_not_found       | Ресурс не найден                                  |
| validation_error         | Ошибка валидации входных данных                   |
| rate_limit_exceeded      | Превышен лимит запросов                           |
| quiz_already_started     | Викторина уже началась                            |
| quiz_not_started         | Викторина еще не началась                         |
| quiz_ended               | Викторина уже завершена                           |

### Обработка ошибок в клиенте

```javascript
function handleApiError(error) {
  switch(error.code) {
    case 'invalid_credentials':
      // Показать сообщение о неверных учетных данных
      showError('Неверное имя пользователя или пароль');
      break;
    case 'token_expired':
    case 'token_invalid':
      // Попытаться обновить токен или перенаправить на страницу входа
      refreshTokenOrRedirect();
      break;
    case 'permission_denied':
      // Показать сообщение об отсутствии прав
      showError('У вас нет прав для выполнения этого действия');
      break;
    case 'resource_not_found':
      // Перенаправить на страницу 404
      redirectTo404Page();
      break;
    case 'validation_error':
      // Показать ошибки валидации
      showValidationErrors(error.details);
      break;
    case 'rate_limit_exceeded':
      // Показать сообщение о превышении лимита
      showError('Превышен лимит запросов. Пожалуйста, попробуйте позже.');
      break;
    default:
      // Общая обработка ошибок
      showError('Произошла ошибка при обработке запроса');
  }
}

// Пример использования
fetch('https://api.triviaserver.com/api/quizzes/123')
  .then(response => response.json())
  .then(data => {
    if (!data.success) {
      handleApiError(data.error);
      return;
    }
    // Обработка успешного ответа
  })
  .catch(error => {
    console.error('Ошибка сети:', error);
    showError('Проблема с подключением к серверу');
  });
```

## Рекомендации по использованию API

1. **Кэширование**: Кэшируйте данные, которые не меняются часто (например, категории викторин)
2. **Обработка ошибок**: Всегда проверяйте поле `success` и обрабатывайте ошибки
3. **Токены**: Храните токены в безопасном месте и обновляйте их своевременно
4. **Rate Limiting**: Учитывайте ограничения API (не более 60 запросов в минуту для большинства эндпоинтов)
5. **Пагинация**: Используйте пагинацию для списков с большим количеством элементов
6. **Фильтрация на сервере**: Используйте параметры фильтрации API вместо фильтрации на клиенте

## Примеры интеграции

### Типичный поток работы с викториной

```javascript
// Класс для работы с API
class TriviaApiClient {
  constructor(baseUrl = 'https://api.triviaserver.com/api') {
    this.baseUrl = baseUrl;
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    const headers = {
      ...(options.headers || {}),
      'Authorization': `Bearer ${localStorage.getItem('access_token')}`
    };

    try {
      const response = await fetch(url, { ...options, headers });
      const data = await response.json();

      if (!data.success) {
        // Обработка ошибок API
        if (data.error.code === 'token_expired') {
          const refreshed = await this.refreshToken();
          if (refreshed) {
            // Повторяем запрос с новым токеном
            return this.request(endpoint, options);
          }
        }
        throw data.error;
      }

      return data.data;
    } catch (error) {
      console.error('API Error:', error);
      throw error;
    }
  }

  async login(email, password) {
    const response = await fetch(`${this.baseUrl}/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, password, token_based: true })
    });

    const data = await response.json();
    if (!data.success) throw data.error;

    localStorage.setItem('access_token', data.data.access_token);
    localStorage.setItem('refresh_token', data.data.refresh_token);
    
    return data.data.user;
  }

  async refreshToken() {
    try {
      const response = await fetch(`${this.baseUrl}/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          refresh_token: localStorage.getItem('refresh_token')
        })
      });

      const data = await response.json();
      if (!data.success) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
        return false;
      }

      localStorage.setItem('access_token', data.data.access_token);
      localStorage.setItem('refresh_token', data.data.refresh_token);
      return true;
    } catch (error) {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      window.location.href = '/login';
      return false;
    }
  }

  // Получение списка викторин
  async getQuizzes(filters = {}, page = 1, perPage = 20) {
    let query = `?page=${page}&per_page=${perPage}`;
    
    // Формирование параметров фильтрации
    Object.entries(filters).forEach(([key, value]) => {
      if (value) query += `&filter[${key}]=${encodeURIComponent(value)}`;
    });
    
    return this.request(`/quizzes${query}`);
  }

  // Получение деталей викторины
  async getQuiz(quizId) {
    return this.request(`/quizzes/${quizId}`);
  }

  // Присоединение к викторине
  async joinQuiz(quizId) {
    return this.request(`/quizzes/${quizId}/join`, { method: 'POST' });
  }

  // Получение результатов викторины
  async getQuizResults(quizId) {
    return this.request(`/quizzes/${quizId}/results`);
  }

  // Получение профиля пользователя
  async getUserProfile() {
    return this.request('/user/profile');
  }

  // Получение списка категорий
  async getCategories() {
    return this.request('/categories');
  }
}

// Пример использования
const api = new TriviaApiClient();

// Авторизация пользователя
async function loginUser() {
  try {
    const user = await api.login('user@example.com', 'password123');
    console.log('Вход выполнен:', user);
    loadDashboard();
  } catch (error) {
    console.error('Ошибка входа:', error);
    showLoginError(error.message);
  }
}

// Загрузка списка викторин
async function loadQuizzes() {
  try {
    const quizzes = await api.getQuizzes({ 
      status: 'upcoming',
      category: 'history'
    });
    
    console.log('Загружены викторины:', quizzes);
    renderQuizList(quizzes);
  } catch (error) {
    console.error('Ошибка загрузки викторин:', error);
  }
}

// Присоединение к викторине
async function joinQuiz(quizId) {
  try {
    const joinResult = await api.joinQuiz(quizId);
    console.log('Присоединение к викторине:', joinResult);
    
    // Перенаправление в комнату ожидания
    if (joinResult.waiting_room_url) {
      window.location.href = joinResult.waiting_room_url;
    }
  } catch (error) {
    console.error('Ошибка при присоединении к викторине:', error);
    showError(error.message);
  }
}
```

Это руководство охватывает основные аспекты работы с REST API Trivia сервиса. Дополнительные сведения о конкретных эндпоинтах и функциональности можно получить, обратившись к полной документации API.