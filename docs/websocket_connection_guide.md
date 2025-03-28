# Руководство по подключению к WebSocket API

## Введение

WebSocket API позволяет создавать интерактивные приложения с обменом данными в реальном времени для Trivia API. WebSocket соединение используется для:

1. Получения обновлений о состоянии викторин в реальном времени
2. Отправки ответов на вопросы
3. Получения уведомлений о событиях аутентификации
4. Поддержания активного соединения с пользовательским интерфейсом

## Базовая информация

### URL подключения

```
wss://api.triviaserver.com/ws
```

Для локальной разработки:

```
ws://localhost:8080/ws
```

### Параметры подключения

При подключении необходимо передать токен аутентификации в URL:

```
wss://api.triviaserver.com/ws?token=ваш_jwt_токен
```

### Формат сообщений

Все сообщения передаются в формате JSON и имеют следующую структуру:

```json
{
  "type": "тип_сообщения",
  "data": { /* данные сообщения */ },
  "priority": "NORMAL" // Опционально: LOW, NORMAL, HIGH, CRITICAL
}
```

## Установление соединения

### Аутентификация

Для установления WebSocket соединения требуется активный JWT токен, полученный при аутентификации через REST API. Токен передается как параметр URL при установлении соединения:

```javascript
const token = "eyJhbGciOiJIUzI1NiIsInR..."; // JWT токен от REST API
const socket = new WebSocket(`wss://api.triviaserver.com/ws?token=${token}`);
```

### Обработка событий подключения

```javascript
socket.onopen = (event) => {
  console.log("WebSocket соединение установлено");
  // Здесь можно отправить сообщение о присоединении к викторине
};

socket.onclose = (event) => {
  // event.code содержит код закрытия соединения
  console.log(`WebSocket соединение закрыто: ${event.code}`, event.reason);
  
  // Коды закрытия:
  // 1000 - нормальное закрытие
  // 1001 - закрытие из-за ухода пользователя со страницы
  // 1006 - аномальное закрытие
  // 4000 - аутентификация не удалась
  // 4001 - токен истек
  // 4002 - недостаточно прав
};

socket.onerror = (error) => {
  console.error("WebSocket ошибка:", error);
};
```

### Переподключение

Рекомендуется реализовать логику автоматического переподключения при обрыве связи:

```javascript
function connectWebSocket() {
  const token = getAuthToken(); // Функция получения текущего токена
  const socket = new WebSocket(`wss://api.triviaserver.com/ws?token=${token}`);
  
  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;
  
  socket.onclose = (event) => {
    console.log(`WebSocket соединение закрыто: ${event.code}`, event.reason);
    
    // Не пытаемся переподключиться при нормальном закрытии
    if (event.code === 1000) return;
    
    // Пытаемся переподключиться при истекшем токене
    if (event.code === 4001) {
      refreshAuthToken() // Функция обновления токена
        .then(() => {
          connectWebSocket(); // Переподключение с новым токеном
        })
        .catch(error => {
          console.error("Не удалось обновить токен:", error);
          // Перенаправление на страницу входа
        });
      return;
    }
    
    // Обычное переподключение при других ошибках
    if (reconnectAttempts < maxReconnectAttempts) {
      reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts - 1), 30000);
      console.log(`Переподключение через ${delay}мс (${reconnectAttempts}/${maxReconnectAttempts})`);
      
      setTimeout(connectWebSocket, delay);
    } else {
      console.error("Достигнуто максимальное количество попыток переподключения");
    }
  };
  
  // Остальные обработчики...
  
  return socket;
}
```

## Отправка и получение сообщений

### Отправка сообщений

Для отправки сообщения серверу используйте метод `send()`:

```javascript
function sendMessage(socket, type, data, priority = "NORMAL") {
  if (socket.readyState !== WebSocket.OPEN) {
    console.error("Попытка отправить сообщение при закрытом соединении");
    return false;
  }
  
  const message = {
    type,
    data,
    priority
  };
  
  socket.send(JSON.stringify(message));
  return true;
}

// Пример использования
sendMessage(socket, "user:join_quiz", { quiz_id: 123 });
```

### Получение сообщений

Для обработки входящих сообщений используйте обработчик `onmessage`:

```javascript
socket.onmessage = (event) => {
  try {
    const message = JSON.parse(event.data);
    console.log("Получено сообщение:", message);
    
    // Обработка сообщения в зависимости от типа
    switch (message.type) {
      case "system:connected":
        handleConnectedMessage(message.data);
        break;
      case "quiz:start":
        handleQuizStart(message.data);
        break;
      case "question:new":
        handleNewQuestion(message.data);
        break;
      case "question:end":
        handleQuestionEnd(message.data);
        break;
      case "quiz:end":
        handleQuizEnd(message.data);
        break;
      case "token:refresh_needed":
        handleTokenRefreshNeeded();
        break;
      case "server:heartbeat":
        handleHeartbeat();
        break;
      default:
        console.log("Неизвестный тип сообщения:", message.type);
    }
  } catch (error) {
    console.error("Ошибка при обработке сообщения:", error);
  }
};
```

## Поддержка соединения (Heartbeat)

Для поддержания активного соединения сервер периодически отправляет heartbeat-сообщения. Клиент должен отвечать на них, чтобы сервер знал, что соединение активно:

```javascript
// Отправка heartbeat каждые 30 секунд
const heartbeatInterval = setInterval(() => {
  if (socket.readyState === WebSocket.OPEN) {
    sendMessage(socket, "user:heartbeat", {});
  }
}, 30000);

// Обработка heartbeat от сервера
function handleHeartbeat() {
  // Обычно никаких действий не требуется,
  // сервер уже получил подтверждение, что клиент активен
  console.log("Heartbeat получен от сервера");
}

// Очистка интервала при закрытии соединения
socket.onclose = (event) => {
  clearInterval(heartbeatInterval);
  // ... остальная логика закрытия
};
```

## Типы сообщений

### Сообщения системы

| Тип сообщения | Направление | Описание |
|---------------|-------------|----------|
| `system:connected` | Сервер → Клиент | Подтверждение успешного соединения |
| `system:error` | Сервер → Клиент | Информация об ошибке |
| `system:shard_migration` | Сервер → Клиент | Уведомление о миграции на другой шард |
| `server:heartbeat` | Сервер → Клиент | Проверка активности соединения |
| `user:heartbeat` | Клиент → Сервер | Ответ на проверку активности |

### Сообщения аутентификации

| Тип сообщения | Направление | Описание |
|---------------|-------------|----------|
| `token:refreshed` | Сервер → Клиент | Токен был обновлен |
| `token:expired` | Сервер → Клиент | Токен истек |
| `token:refresh_needed` | Сервер → Клиент | Токен скоро истечет, нужно обновить |
| `token:revoked` | Сервер → Клиент | Токен был отозван |
| `token:invalidated` | Сервер → Клиент | Токен стал недействительным |
| `token:key_rotation` | Сервер → Клиент | Произошла ротация ключей, нужно обновить токен |

### Сообщения викторин

| Тип сообщения | Направление | Описание |
|---------------|-------------|----------|
| `user:join_quiz` | Клиент → Сервер | Запрос на присоединение к викторине |
| `user:leave_quiz` | Клиент → Сервер | Запрос на выход из викторины |
| `quiz:join_confirmed` | Сервер → Клиент | Подтверждение присоединения к викторине |
| `quiz:user_joined` | Сервер → Клиент | Новый пользователь присоединился к викторине |
| `quiz:user_left` | Сервер → Клиент | Пользователь покинул викторину |
| `quiz:start` | Сервер → Клиент | Викторина началась |
| `quiz:end` | Сервер → Клиент | Викторина завершилась |
| `quiz:cancelled` | Сервер → Клиент | Викторина отменена |
| `quiz:postponed` | Сервер → Клиент | Викторина отложена |
| `quiz:starting_soon` | Сервер → Клиент | Викторина скоро начнется |
| `quiz:participants_update` | Сервер → Клиент | Обновление количества участников |

### Сообщения вопросов и ответов

| Тип сообщения | Направление | Описание |
|---------------|-------------|----------|
| `question:new` | Сервер → Клиент | Новый вопрос |
| `question:end` | Сервер → Клиент | Окончание времени на вопрос |
| `user:answer` | Клиент → Сервер | Ответ пользователя на вопрос |
| `answer:received` | Сервер → Клиент | Подтверждение получения ответа |
| `answer:results` | Сервер → Клиент | Результаты ответа |
| `leaderboard:update` | Сервер → Клиент | Обновление таблицы лидеров |

## Примеры

### Полный пример установки соединения и обработки сообщений

```javascript
class TriviaWebSocketClient {
  constructor(apiBaseUrl) {
    this.apiBaseUrl = apiBaseUrl;
    this.socket = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.heartbeatInterval = null;
    this.eventHandlers = {};
    this.isConnecting = false;
  }
  
  connect() {
    if (this.isConnecting) return Promise.reject(new Error("Соединение уже устанавливается"));
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      return Promise.resolve(this.socket);
    }
    
    this.isConnecting = true;
    
    return new Promise((resolve, reject) => {
      try {
        const token = localStorage.getItem("auth_token");
        if (!token) {
          this.isConnecting = false;
          return reject(new Error("Отсутствует токен аутентификации"));
        }
        
        this.socket = new WebSocket(`${this.apiBaseUrl}/ws?token=${token}`);
        
        this.socket.onopen = (event) => {
          console.log("WebSocket соединение установлено");
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          this.isConnecting = false;
          resolve(this.socket);
          this.triggerEvent("connected");
        };
        
        this.socket.onclose = (event) => {
          console.log(`WebSocket соединение закрыто: ${event.code}`, event.reason);
          this.stopHeartbeat();
          this.isConnecting = false;
          this.triggerEvent("disconnected", { code: event.code, reason: event.reason });
          
          // Не переподключаемся при нормальном закрытии
          if (event.code === 1000) return;
          
          // Обновление токена при ошибке аутентификации
          if (event.code === 4001) {
            this.refreshToken()
              .then(() => this.connect())
              .catch(error => {
                console.error("Ошибка обновления токена:", error);
                reject(error);
              });
            return;
          }
          
          this.handleReconnect();
        };
        
        this.socket.onerror = (error) => {
          console.error("WebSocket ошибка:", error);
          this.triggerEvent("error", error);
        };
        
        this.socket.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            console.log("Получено сообщение:", message);
            
            // Обработка системных сообщений
            if (message.type === "server:heartbeat") {
              this.handleHeartbeat();
              return;
            }
            
            if (message.type === "token:refresh_needed") {
              this.refreshToken().catch(console.error);
              return;
            }
            
            // Триггер события по типу сообщения
            this.triggerEvent(message.type, message.data);
            
            // Триггер общего события для всех сообщений
            this.triggerEvent("message", message);
          } catch (error) {
            console.error("Ошибка при обработке сообщения:", error);
          }
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }
  
  disconnect() {
    if (this.socket) {
      this.socket.close(1000, "Нормальное закрытие");
      this.socket = null;
    }
    this.stopHeartbeat();
  }
  
  sendMessage(type, data, priority = "NORMAL") {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error("Попытка отправить сообщение при закрытом соединении");
      return false;
    }
    
    const message = {
      type,
      data,
      priority
    };
    
    this.socket.send(JSON.stringify(message));
    return true;
  }
  
  on(eventType, handler) {
    if (!this.eventHandlers[eventType]) {
      this.eventHandlers[eventType] = [];
    }
    this.eventHandlers[eventType].push(handler);
    return this;
  }
  
  off(eventType, handler) {
    if (!this.eventHandlers[eventType]) return this;
    
    if (handler) {
      this.eventHandlers[eventType] = this.eventHandlers[eventType].filter(
        h => h !== handler
      );
    } else {
      this.eventHandlers[eventType] = [];
    }
    
    return this;
  }
  
  triggerEvent(eventType, data) {
    if (!this.eventHandlers[eventType]) return;
    
    for (const handler of this.eventHandlers[eventType]) {
      try {
        handler(data);
      } catch (error) {
        console.error(`Ошибка в обработчике события ${eventType}:`, error);
      }
    }
  }
  
  startHeartbeat() {
    this.stopHeartbeat();
    
    this.heartbeatInterval = setInterval(() => {
      if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        this.sendMessage("user:heartbeat", {});
      }
    }, 30000);
  }
  
  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
  
  handleHeartbeat() {
    console.log("Heartbeat получен от сервера");
  }
  
  handleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("Достигнуто максимальное количество попыток переподключения");
      return;
    }
    
    this.reconnectAttempts++;
    
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts - 1),
      30000
    );
    
    console.log(`Переподключение через ${delay}мс (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
    
    setTimeout(() => {
      this.connect().catch(console.error);
    }, delay);
  }
  
  refreshToken() {
    return fetch(`${this.apiBaseUrl}/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        refresh_token: localStorage.getItem("refresh_token")
      })
    })
    .then(response => response.json())
    .then(data => {
      if (!data.success) {
        throw new Error(data.error || "Не удалось обновить токен");
      }
      
      localStorage.setItem("auth_token", data.data.access_token);
      localStorage.setItem("refresh_token", data.data.refresh_token);
      
      return data.data;
    });
  }
  
  // Вспомогательные методы для викторин
  
  joinQuiz(quizId) {
    return this.sendMessage("user:join_quiz", { quiz_id: quizId });
  }
  
  leaveQuiz(quizId) {
    return this.sendMessage("user:leave_quiz", { quiz_id: quizId });
  }
  
  sendAnswer(questionId, selectedOptionId, answerTimeMs) {
    return this.sendMessage("user:answer", {
      question_id: questionId,
      selected_option: selectedOptionId,
      answer_time_ms: answerTimeMs
    });
  }
}

// Пример использования
const triviaClient = new TriviaWebSocketClient("wss://api.triviaserver.com");

triviaClient.on("connected", () => {
  console.log("Подключено к WebSocket серверу");
  // Присоединиться к викторине после подключения
  triviaClient.joinQuiz(123);
});

triviaClient.on("quiz:join_confirmed", (data) => {
  console.log(`Присоединился к викторине ${data.quiz_id}`);
});

triviaClient.on("quiz:start", (data) => {
  console.log("Викторина началась!", data);
});

triviaClient.on("question:new", (data) => {
  console.log("Новый вопрос:", data);
  // Отобразить вопрос в пользовательском интерфейсе
});

triviaClient.on("question:end", (data) => {
  console.log("Время на вопрос истекло:", data);
});

triviaClient.on("answer:results", (data) => {
  console.log("Результаты ответа:", data);
});

triviaClient.on("quiz:end", (data) => {
  console.log("Викторина завершена!", data);
});

triviaClient.on("disconnected", (data) => {
  console.log("Отключено от WebSocket сервера", data);
});

// Подключиться к серверу
triviaClient.connect().catch(error => {
  console.error("Ошибка подключения:", error);
});
```

### Пример обработки викторины в React

```jsx
import React, { useEffect, useState } from 'react';
import TriviaWebSocketClient from '../services/TriviaWebSocketClient';

const QuizPage = ({ quizId }) => {
  const [client] = useState(() => new TriviaWebSocketClient("wss://api.triviaserver.com"));
  const [status, setStatus] = useState('waiting');
  const [currentQuestion, setCurrentQuestion] = useState(null);
  const [remainingTime, setRemainingTime] = useState(0);
  const [timerInterval, setTimerInterval] = useState(null);
  const [results, setResults] = useState(null);
  const [selectedOption, setSelectedOption] = useState(null);
  const [answerSent, setAnswerSent] = useState(false);
  const [questionStartTime, setQuestionStartTime] = useState(null);
  
  useEffect(() => {
    // Обработчики событий WebSocket
    client.on("connected", () => {
      client.joinQuiz(quizId);
    });
    
    client.on("quiz:join_confirmed", (data) => {
      setStatus('waiting');
    });
    
    client.on("quiz:start", (data) => {
      setStatus('active');
      setResults(null);
    });
    
    client.on("question:new", (data) => {
      setCurrentQuestion(data);
      setSelectedOption(null);
      setAnswerSent(false);
      setQuestionStartTime(Date.now());
      setRemainingTime(data.time_limit_sec);
      
      // Запустить таймер
      if (timerInterval) clearInterval(timerInterval);
      
      const interval = setInterval(() => {
        setRemainingTime((prevTime) => {
          if (prevTime <= 1) {
            clearInterval(interval);
            return 0;
          }
          return prevTime - 1;
        });
      }, 1000);
      
      setTimerInterval(interval);
    });
    
    client.on("question:end", (data) => {
      if (timerInterval) clearInterval(timerInterval);
      setRemainingTime(0);
      
      // Если пользователь не успел ответить, отправить пустой ответ
      if (!answerSent && currentQuestion) {
        sendAnswer(null);
      }
    });
    
    client.on("answer:results", (data) => {
      // Обновить результаты текущего вопроса
    });
    
    client.on("quiz:end", (data) => {
      setStatus('completed');
      setResults(data.results);
      setCurrentQuestion(null);
      if (timerInterval) clearInterval(timerInterval);
    });
    
    client.on("quiz:cancelled", () => {
      setStatus('cancelled');
      setCurrentQuestion(null);
      if (timerInterval) clearInterval(timerInterval);
    });
    
    // Подключиться к WebSocket
    client.connect().catch(console.error);
    
    // Очистка при размонтировании
    return () => {
      client.off("connected");
      client.off("quiz:join_confirmed");
      client.off("quiz:start");
      client.off("question:new");
      client.off("question:end");
      client.off("answer:results");
      client.off("quiz:end");
      client.off("quiz:cancelled");
      
      if (timerInterval) clearInterval(timerInterval);
      
      client.leaveQuiz(quizId);
    };
  }, [client, quizId, timerInterval, answerSent, currentQuestion]);
  
  const sendAnswer = (optionId) => {
    if (!currentQuestion || answerSent) return;
    
    const answerTimeMs = Date.now() - questionStartTime;
    client.sendAnswer(currentQuestion.id, optionId, answerTimeMs);
    setSelectedOption(optionId);
    setAnswerSent(true);
  };
  
  // Рендеринг в зависимости от состояния
  if (status === 'waiting') {
    return <div>Ожидание начала викторины...</div>;
  }
  
  if (status === 'active' && currentQuestion) {
    return (
      <div>
        <div className="timer">Осталось времени: {remainingTime}с</div>
        <h2>{currentQuestion.text}</h2>
        <div className="options">
          {currentQuestion.options.map((option) => (
            <button
              key={option.id}
              onClick={() => sendAnswer(option.id)}
              disabled={answerSent}
              className={selectedOption === option.id ? 'selected' : ''}
            >
              {option.text}
            </button>
          ))}
        </div>
      </div>
    );
  }
  
  if (status === 'completed' && results) {
    return (
      <div>
        <h2>Викторина завершена!</h2>
        <div>Ваш счет: {results.score}</div>
        <div>Правильных ответов: {results.correct_answers}/{results.total_questions}</div>
        <div>Место в рейтинге: {results.rank}</div>
      </div>
    );
  }
  
  if (status === 'cancelled') {
    return <div>Викторина была отменена.</div>;
  }
  
  return <div>Загрузка...</div>;
};

export default QuizPage;
```

## Расширенные возможности

### Приоритеты сообщений

Для обеспечения более гибкого взаимодействия WebSocket API поддерживает приоритеты сообщений:

1. `LOW` - низкий приоритет, для некритичных обновлений
2. `NORMAL` - стандартный приоритет (по умолчанию)
3. `HIGH` - высокий приоритет, для важных сообщений
4. `CRITICAL` - критический приоритет, обрабатывается в первую очередь

Пример отправки сообщения с высоким приоритетом:

```javascript
client.sendMessage("user:answer", {
  question_id: 123,
  selected_option: 2,
  answer_time_ms: 5230
}, "HIGH");
```

### Обработка ошибок

WebSocket API может возвращать сообщения об ошибках с типом `system:error`:

```json
{
  "type": "system:error",
  "data": {
    "code": "invalid_message_format",
    "message": "Неверный формат сообщения"
  }
}
```

Основные коды ошибок:

| Код ошибки | Описание |
|------------|----------|
| `invalid_message_format` | Неверный формат сообщения |
| `invalid_message_type` | Неизвестный тип сообщения |
| `invalid_message_data` | Некорректные данные сообщения |
| `forbidden` | Недостаточно прав для выполнения операции |
| `not_found` | Запрашиваемый ресурс не найден |
| `not_available` | Ресурс временно недоступен |
| `rate_limited` | Превышен лимит запросов |

### Сжатие данных

Для оптимизации трафика WebSocket API поддерживает сжатие сообщений. Для включения сжатия, укажите параметр `compression=true` при установлении соединения:

```javascript
const socket = new WebSocket(`wss://api.triviaserver.com/ws?token=${token}&compression=true`);
```

## Заключение

WebSocket API Trivia API предоставляет мощный механизм для создания интерактивных викторин в реальном времени. Следуя рекомендациям этого руководства, вы сможете реализовать надежное и эффективное взаимодействие клиента с сервером.

Основные рекомендации:
1. Всегда реализуйте надежную логику переподключения
2. Регулярно отправляйте и обрабатывайте heartbeat-сообщения
3. Обрабатывайте все типы ошибок и уведомлений о токенах
4. Структурируйте код для удобной обработки всех типов сообщений
5. Не забывайте очищать ресурсы (интервалы, обработчики) при закрытии соединения 