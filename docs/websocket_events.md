# Документация по WebSocket-событиям

## Обзор

В этом документе представлена полная документация по всем типам событий, передаваемых через WebSocket в Trivia API. Документ синхронизирует типы событий, используемые на бэкенде и фронтенде, обеспечивая единое понимание системы всеми компонентами.

## Типы событий

### События, связанные с токенами авторизации

| Тип события | Источник | Описание | Приоритет | Структура данных |
|-------------|----------|----------|-----------|------------------|
| `token_refreshed` | Бэкенд → Фронтенд | Уведомление об успешном обновлении токена | HIGH | `TokenRefreshedEvent` |
| `token_expired` | Бэкенд → Фронтенд | Уведомление об истечении срока действия токена | CRITICAL | `TokenInvalidatedEvent` |
| `token_revoked` | Бэкенд → Фронтенд | Уведомление об отзыве токена (например, выход с других устройств) | CRITICAL | `TokenInvalidatedEvent` |
| `token_invalidated` | Бэкенд → Фронтенд | Уведомление о недействительности токена по другим причинам | CRITICAL | `TokenInvalidatedEvent` |
| `token_about_to_expire` | Бэкенд → Фронтенд | Предупреждение о скором истечении срока действия токена | HIGH | `TokenExpiryWarningEvent` |
| `key_rotation` | Бэкенд → Фронтенд | Уведомление о ротации ключей JWT и необходимости обновления токенов | HIGH | `KeyRotationEvent` |

### События викторины

| Тип события | Источник | Описание | Приоритет | Структура данных |
|-------------|----------|----------|-----------|------------------|
| `QUIZ_START` | Бэкенд → Фронтенд | Уведомление о начале викторины | HIGH | `QuizStartEvent` |
| `QUIZ_END` | Бэкенд → Фронтенд | Уведомление о завершении викторины | HIGH | `QuizEndEvent` |
| `QUESTION_START` | Бэкенд → Фронтенд | Уведомление о начале нового вопроса | HIGH | `QuestionStartEvent` |
| `QUESTION_END` | Бэкенд → Фронтенд | Уведомление о завершении текущего вопроса | HIGH | `QuestionEndEvent` |
| `USER_ANSWER` | Фронтенд → Бэкенд | Отправка ответа пользователя | NORMAL | `UserAnswerEvent` |
| `RESULT_UPDATE` | Бэкенд → Фронтенд | Обновление результатов | NORMAL | `ResultUpdateEvent` |

### Системные события

| Тип события | Источник | Описание | Приоритет | Структура данных |
|-------------|----------|----------|-----------|------------------|
| `USER_HEARTBEAT` | Фронтенд → Бэкенд | Проверка активности соединения | LOW | - |
| `SERVER_HEARTBEAT` | Бэкенд → Фронтенд | Ответ на проверку активности | LOW | - |
| `USER_DISCONNECT` | Фронтенд → Бэкенд | Уведомление о намерении отключиться | HIGH | - |
| `SHARD_MIGRATION` | Бэкенд → Фронтенд | Уведомление о миграции на другой шард | CRITICAL | `ShardMigrationEvent` |

## Структуры данных событий

### События токенов

#### TokenRefreshedEvent
```typescript
interface TokenRefreshedEvent {
  user_id: number;
  device_id?: string;
  access_token: string;
  csrf_token: string;
  expires_in: number;
}
```

#### TokenInvalidatedEvent
```typescript
interface TokenInvalidatedEvent {
  user_id: number;
  device_id?: string;
  token_id?: string;
  reason: string;
}
```

#### TokenExpiryWarningEvent
```typescript
interface TokenExpiryWarningEvent {
  user_id: number;
  expires_in: number; // секунды до истечения
  token_id?: string;
}
```

#### KeyRotationEvent
```typescript
interface KeyRotationEvent {
  user_id: number;
  device_id?: string;
  access_token: string;
  csrf_token: string;
  expires_in: number;
  rotation_reason?: string; // Причина ротации (плановая, внеплановая и т.д.)
}
```

### События викторины

#### QuizStartEvent
```typescript
interface QuizStartEvent {
  quiz_id: number;
  title: string;
  description: string;
  num_questions: number;
  duration_minutes: number;
  start_time: string; // ISO 8601 формат даты
}
```

#### QuestionStartEvent
```typescript
interface QuestionStartEvent {
  quiz_id: number;
  question_id: number;
  question_number: number;
  text: string;
  options: Array<{
    id: number;
    text: string;
  }>;
  duration_seconds: number;
  start_time: string; // ISO 8601 формат даты
}
```

#### UserAnswerEvent
```typescript
interface UserAnswerEvent {
  quiz_id: number;
  question_id: number;
  option_id: number;
  answer_time: string; // ISO 8601 формат даты
}
```

#### ResultUpdateEvent
```typescript
interface ResultUpdateEvent {
  quiz_id: number;
  leaderboard: Array<{
    user_id: number;
    username: string;
    score: number;
    position: number;
  }>;
  user_stats?: {
    correct_answers: number;
    total_answers: number;
    position: number;
    score: number;
  };
}
```

### Системные события

#### ShardMigrationEvent
```typescript
interface ShardMigrationEvent {
  old_shard_id: number;
  new_shard_id: number;
  migration_token: string;
  migration_reason: string;
}
```

## Приоритеты сообщений

| Приоритет | Числовое значение | Описание |
|-----------|------------------|----------|
| CRITICAL | 3 | Критичные системные сообщения, требующие немедленной обработки |
| HIGH | 2 | Важные сообщения, обрабатываемые с высоким приоритетом |
| NORMAL | 1 | Стандартные сообщения (по умолчанию) |
| LOW | 0 | Некритичные сообщения, которые могут быть отложены |

## Работа с событиями на фронтенде

### Подписка на события
```typescript
// Подписка на событие обновления токена
WebSocketService.addMessageHandler(TokenEventType.TOKEN_REFRESHED, 
  (data: TokenRefreshedEvent) => {
    // Обработка обновления токена
  }
);
```

### Отправка событий
```typescript
// Отправка ответа пользователя
WebSocketService.sendMessage({
  type: 'USER_ANSWER',
  data: {
    quiz_id: 123,
    question_id: 456,
    option_id: 2,
    answer_time: new Date().toISOString()
  },
  priority: MessagePriority.NORMAL
});
```

## Работа с событиями на бэкенде

### Отправка события конкретному пользователю
```go
// Отправка предупреждения о скором истечении токена
func (m *Manager) SendTokenExpirationWarning(userID string, expiresIn int) {
    message := map[string]interface{}{
        "type": TOKEN_EXPIRE_SOON,
        "data": map[string]interface{}{
            "expires_in": expiresIn,
            "unit":       "seconds",
        },
    }
    
    jsonMessage, _ := json.Marshal(message)
    m.hub.SendToUser(userID, jsonMessage)
}
```

### Широковещательная отправка
```go
// Отправка сообщения о начале вопроса всем подписчикам
func (h *QuizHandler) startQuestion(quizID int, questionData QuestionStartEvent) {
    event := map[string]interface{}{
        "type": "QUESTION_START",
        "data": questionData,
    }
    
    h.wsManager.BroadcastEvent("quiz:"+strconv.Itoa(quizID), event)
}
```

## Соответствие между фронтендом и бэкендом

| Фронтенд константа | Бэкенд константа |
|--------------------|------------------|
| `TokenEventType.TOKEN_REFRESHED` | `"token_refreshed"` |
| `TokenEventType.TOKEN_EXPIRED` | `TOKEN_EXPIRED` |
| `TokenEventType.TOKEN_REVOKED` | `"token_revoked"` |
| `TokenEventType.TOKEN_INVALIDATED` | `"token_invalidated"` |
| `TokenEventType.TOKEN_ABOUT_TO_EXPIRE` | `TOKEN_EXPIRE_SOON` |
| `TokenEventType.KEY_ROTATION` | `"key_rotation"` |
| `WebSocketEventType.QUIZ_START` | `QUIZ_START` |
| `WebSocketEventType.QUIZ_END` | `QUIZ_END` |
| `WebSocketEventType.QUESTION_START` | `QUESTION_START` |
| `WebSocketEventType.QUESTION_END` | `QUESTION_END` |
| `WebSocketEventType.USER_ANSWER` | `USER_ANSWER` |
| `WebSocketEventType.RESULT_UPDATE` | `RESULT_UPDATE` |

## Рекомендации по имплементации

1. **Унификация констант**: Рекомендуется унифицировать константы на бэкенде и фронтенде, используя одинаковые имена для одинаковых типов событий.

2. **Автогенерация типов**: Рассмотрите возможность автоматической генерации TypeScript-типов из Go-структур для обеспечения согласованности.

3. **Версионирование событий**: При изменении структуры данных события, рекомендуется добавлять номер версии, чтобы обеспечить обратную совместимость.

4. **Документирование приоритетов**: Каждое событие должно иметь документированный приоритет обработки на стороне клиента и сервера.

5. **Валидация схемы**: Реализуйте валидацию схемы сообщений на обеих сторонах для обеспечения согласованности данных. 