# Структура API и WebSocket событий для бэкенда Trivia Quiz

## REST API Endpoints

### Аутентификация
- `POST /api/auth/register` - Регистрация нового пользователя
  - Тело запроса: `{ "username": string, "email": string, "password": string }`
  - Ответ: `{ "token": string, "user": { "id": number, "username": string, "email": string, ... } }`

- `POST /api/auth/login` - Вход в систему
  - Тело запроса: `{ "email": string, "password": string }`
  - Ответ: `{ "token": string, "user": { "id": number, "username": string, "email": string, ... } }`

### Пользователи
- `GET /api/users/me` - Получение данных текущего пользователя
  - Заголовок: `Authorization: Bearer {token}`
  - Ответ: `{ "id": number, "username": string, "email": string, "profile_picture": string, ... }`

- `PUT /api/users/me` - Обновление данных пользователя
  - Заголовок: `Authorization: Bearer {token}`
  - Тело запроса: `{ "username": string, "profile_picture": string }`
  - Ответ: `{ "message": "Profile updated successfully" }`

### Викторины
- `GET /api/quizzes` - Список всех викторин с пагинацией
  - Параметры запроса: `page`, `page_size`
  - Ответ: `[{ "id": number, "title": string, "description": string, "scheduled_time": string, "status": string, ... }, ...]`

- `GET /api/quizzes/active` - Получение активной викторины
  - Ответ: `{ "id": number, "title": string, "description": string, ... }` или `{ "error": "No active quiz" }`

- `GET /api/quizzes/scheduled` - Получение запланированных викторин
  - Ответ: `[{ "id": number, "title": string, "scheduled_time": string, ... }, ...]`

- `GET /api/quizzes/:id` - Получение детальной информации о викторине
  - Ответ: `{ "id": number, "title": string, "description": string, "status": string, ... }`

- `GET /api/quizzes/:id/with-questions` - Получение викторины с вопросами
  - Ответ: `{ "id": number, "title": string, "questions": [{ "id": number, "text": string, "options": [string, ...], ... }, ...], ... }`

- `GET /api/quizzes/:id/results` - Получение результатов викторины
  - Ответ: `[{ "user_id": number, "username": string, "score": number, "rank": number, ... }, ...]`

- `GET /api/quizzes/:id/my-result` - Получение результата текущего пользователя
  - Заголовок: `Authorization: Bearer {token}`
  - Ответ: `{ "score": number, "correct_answers": number, "rank": number, ... }`

### Администрирование викторин
- `POST /api/quizzes` - Создание новой викторины
  - Заголовок: `Authorization: Bearer {token}`
  - Тело запроса: `{ "title": string, "description": string, "scheduled_time": string }`
  - Ответ: `{ "id": number, "title": string, ... }`

- `POST /api/quizzes/:id/questions` - Добавление вопросов к викторине
  - Заголовок: `Authorization: Bearer {token}`
  - Тело запроса: `{ "questions": [{ "text": string, "options": [string, ...], "correct_option": number, "time_limit_sec": number, "point_value": number }, ...] }`
  - Ответ: `{ "message": "Questions added successfully" }`

- `PUT /api/quizzes/:id/schedule` - Планирование времени викторины
  - Заголовок: `Authorization: Bearer {token}`
  - Тело запроса: `{ "scheduled_time": string }`
  - Ответ: `{ "message": "Quiz scheduled successfully" }`

- `PUT /api/quizzes/:id/cancel` - Отмена викторины
  - Заголовок: `Authorization: Bearer {token}`
  - Ответ: `{ "message": "Quiz cancelled successfully" }`

## WebSocket API

### Соединение
- `GET /ws?token={jwt_token}` - Подключение к WebSocket

### События от клиента к серверу
- `user:ready` - Пользователь готов к викторине
  ```json
  {
    "type": "user:ready",
    "data": {
      "quiz_id": number
    }
  }
  ```

- `user:answer` - Ответ пользователя на вопрос
  ```json
  {
    "type": "user:answer",
    "data": {
      "question_id": number,
      "selected_option": number,
      "timestamp": number // миллисекунды
    }
  }
  ```

- `user:heartbeat` - Проверка соединения
  ```json
  {
    "type": "user:heartbeat",
    "data": {}
  }
  ```

### События от сервера к клиенту
- `quiz:announcement` - Анонс викторины (за 30 минут)
  ```json
  {
    "type": "quiz:announcement",
    "data": {
      "quiz_id": number,
      "title": string,
      "description": string,
      "scheduled_time": string,
      "question_count": number
    }
  }
  ```

- `quiz:waiting_room` - Открытие зала ожидания (за 5 минут)
  ```json
  {
    "type": "quiz:waiting_room",
    "data": {
      "quiz_id": number,
      "title": string,
      "starts_in_seconds": number
    }
  }
  ```

- `quiz:countdown` - Обратный отсчет (за 1 минуту)
  ```json
  {
    "type": "quiz:countdown",
    "data": {
      "quiz_id": number,
      "seconds_left": number
    }
  }
  ```

- `quiz:start` - Начало викторины
  ```json
  {
    "type": "quiz:start",
    "data": {
      "quiz_id": number,
      "title": string,
      "question_count": number
    }
  }
  ```

- `quiz:question` - Новый вопрос
  ```json
  {
    "type": "quiz:question",
    "data": {
      "question_id": number,
      "quiz_id": number,
      "number": number,
      "text": string,
      "options": [string, ...],
      "time_limit": number,
      "total_questions": number
    }
  }
  ```

- `quiz:timer` - Обновление таймера
  ```json
  {
    "type": "quiz:timer",
    "data": {
      "question_id": number,
      "remaining_seconds": number
    }
  }
  ```

- `quiz:answer_reveal` - Показ правильного ответа
  ```json
  {
    "type": "quiz:answer_reveal",
    "data": {
      "question_id": number,
      "correct_option": number
    }
  }
  ```

- `quiz:answer_result` - Результат ответа пользователя
  ```json
  {
    "type": "quiz:answer_result",
    "data": {
      "question_id": number,
      "correct_option": number,
      "your_answer": number,
      "is_correct": boolean,
      "points_earned": number,
      "time_taken_ms": number
    }
  }
  ```

- `quiz:end` - Конец викторины
  ```json
  {
    "type": "quiz:end",
    "data": {
      "quiz_id": number,
      "message": "Quiz has ended"
    }
  }
  ```

- `quiz:leaderboard` - Таблица лидеров
  ```json
  {
    "type": "quiz:leaderboard",
    "data": {
      "quiz_id": number,
      "results": [
        {
          "user_id": number,
          "username": string,
          "score": number,
          "correct_answers": number,
          "rank": number
        },
        ...
      ]
    }
  }
  ```

- `quiz:user_ready` - Уведомление о готовности пользователя
  ```json
  {
    "type": "quiz:user_ready",
    "data": {
      "user_id": number,
      "quiz_id": number,
      "status": "ready"
    }
  }
  ```

- `quiz:cancelled` - Уведомление об отмене викторины
  ```json
  {
    "type": "quiz:cancelled",
    "data": {
      "quiz_id": number,
      "message": "Quiz has been cancelled"
    }
  }
  ```

- `server:heartbeat` - Ответ на проверку соединения
  ```json
  {
    "type": "server:heartbeat",
    "data": {
      "timestamp": number
    }
  }
  ```

- `error` - Сообщение об ошибке
  ```json
  {
    "type": "error",
    "data": {
      "message": string
    }
  }
  ```
