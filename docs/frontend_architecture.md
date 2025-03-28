Руководство по архитектуре фронтенда (React + Next.js)
Обзор
Данное руководство содержит рекомендации по построению архитектуры фронтенд-приложения на стеке React/Next.js/TypeScript для взаимодействия с Trivia API. Документ предлагает оптимальные подходы к организации кода, управлению состоянием, кэшированию данных и взаимодействию с API, учитывая особенности бэкенда (включая двойной режим аутентификации и WebSocket).

Рекомендуемый стек технологий
Trivia API оптимально работает с данным стеком:

Основа приложения
React – Библиотека для создания пользовательского интерфейса.

TypeScript – Для статической типизации и повышения надежности кода.

Next.js – React-фреймворк для SSR/SSG, статической экспорт, организации роутинга, API routes и оптимизаций.

Управление состоянием
React Query (TanStack Query) – Для управления серверным состоянием: запросы к API, кэширование, инвалидация, фоновое обновление.

Redux Toolkit – Для управления глобальным состоянием UI, не связанным напрямую с сервером (например, состояние аутентификации (включая флаг useCookieAuth, CSRF-токен), статус WebSocket соединения, состояние UI-элементов).

Компоненты интерфейса
TailwindCSS – Утилитарный CSS-фреймворк для быстрой стилизации компонентов.

Headless UI / Radix UI – Для создания базовых, доступных и кастомизируемых UI-компонетов (модальные окна, выпадающие списки и т.д.).

Framer Motion – Для создания плавных анимаций в React-приложениях.

Коммуникация с API
Axios – HTTP-клиент для взаимодействия с REST API бэкенда.

Кастомный WebSocket клиент (на основе нативного API) – Обертка для управления WebSocket соединением (переподключение, heartbeat, обработка сообщений).

Структура проекта
Рекомендуемая структура директорий для Next.js проекта:

src/
  ├── api/                 # API клиенты и сервисы
  │   ├── http/            # Axios клиент и интерцепторы
  │   ├── websocket/       # WebSocket клиент и обработчики
  │   └── services/        # Сервисы (функции) для работы с API эндпоинтами
  ├── components/          # React компоненты
  │   ├── common/          # Общие переиспользуемые компоненты (кнопки, инпуты, спиннеры)
  │   ├── quiz/            # Компоненты, специфичные для викторин
  │   ├── auth/            # Компоненты аутентификации (формы логина/регистрации)
  │   └── layout/          # Компоненты макета страницы (Header, Footer, Sidebar)
  ├── hooks/               # Кастомные React хуки (включая хуки для React Query)
  ├── pages/               # Страницы приложения (роутинг Next.js)
  │   ├── api/             # API Routes Next.js (если необходимы)
  │   ├── _app.tsx         # Глобальный компонент приложения
  │   └── _document.tsx    # Кастомизация HTML документа
  ├── store/               # Хранилище состояния Redux Toolkit
  │   ├── index.ts         # Конфигурация стора
  │   ├── auth/            # Слайс и действия для аутентификации
  │   ├── quiz/            # Слайс и действия для состояния викторины (управляемого WS)
  │   └── websocket/       # Слайс и действия для состояния WebSocket
  ├── types/               # Глобальные TypeScript типы и интерфейсы (API модели и т.д.)
  ├── utils/               # Вспомогательные функции (форматирование дат, валидация и т.п.)
  ├── constants/           # Константы приложения (ключи API, роуты и т.п.)
  └── public/              # Статические ресурсы (изображения, шрифты)
styles/                  # Глобальные стили (если нужны)
Use code with caution.
Примечание: pages/ директория используется Next.js для файлового роутинга.

Модель данных
Ключевые TypeScript типы данных, соответствующие моделям Go бэкенда:

// Пользователь (src/types/user.ts)
export interface User {
  id: number;
  username: string;
  email: string;
  profile_picture?: string; // Соответствует avatar_url в старом описании? Уточнить по бэкенду
  // role: 'user' | 'admin' | 'moderator'; // Уточнить, есть ли роль на бэкенде
  games_played: number;
  total_score: number;
  highest_score: number;
  created_at: string;
  updated_at: string;
}

// Викторина (src/types/quiz.ts)
export interface Quiz {
  id: number;
  title: string;
  description: string;
  // category: string; // Уточнить, есть ли на бэкенде
  // difficulty: 'easy' | 'medium' | 'hard'; // Уточнить, есть ли на бэкенде
  // creator_id: number; // Уточнить, есть ли на бэкенде
  // is_public: boolean; // Уточнить, есть ли на бэкенде
  scheduled_time: string;
  // duration_minutes: number; // Уточнить, есть ли на бэкенде
  question_count: number;
  status: 'scheduled' | 'in_progress' | 'completed' | 'cancelled'; // 'active' переименован в 'in_progress' на бэкенде? 'draft' отсутствует?
  created_at: string;
  updated_at: string;
}

// Вопрос (src/types/question.ts)
export interface Question {
  id: number;
  quiz_id: number;
  text: string;
  options: Array<{ id: number; text: string }>; // Структура options уточнена? На бэкенде JSONB
  correct_option: number; // Индекс правильного ответа
  time_limit_sec: number;
  point_value: number;
}

// Ответ пользователя (src/types/answer.ts)
export interface UserAnswer {
  id: number; // Добавлено ID из схемы БД
  user_id: number;
  quiz_id: number;
  question_id: number;
  selected_option: number;
  is_correct: boolean;
  response_time_ms: number; // Соответствует answer_time_ms?
  score: number; // Соответствует points_earned?
  created_at: string; // Соответствует submitted_at?
}

// Результат викторины пользователя (src/types/result.ts)
export interface UserQuizResult {
  id: number; // Добавлено ID из схемы БД
  user_id: number;
  quiz_id: number;
  username: string;
  profile_picture?: string;
  score: number;
  correct_answers: number;
  total_questions: number;
  rank: number;
  completed_at: string;
  created_at: string;
}

// WebSocket сообщение (src/types/websocket.ts)
export interface WebSocketMessage {
  type: string; // Например: 'QUIZ_START', 'QUESTION_START', 'USER_ANSWER', 'RESULT_UPDATE'
  data: any;
  priority?: 'LOW' | 'NORMAL' | 'HIGH' | 'CRITICAL'; // Если бэкенд отправляет
}

// Типы для Auth Slice (src/store/auth/types.ts)
export interface AuthState {
  user: User | null;
  token: string | null; // Access Token (используется при Bearer Auth)
  csrfToken: string | null; // CSRF Token (используется при Cookie Auth)
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  useCookieAuth: boolean; // Флаг для выбора режима аутентификации
}

// Типы для Quiz Slice (src/store/quiz/types.ts)
export interface QuizState {
  activeQuiz: Quiz | null; // Информация о текущей активной викторине
  currentQuestion: Question | null; // Текущий вопрос (полученный по WS)
  userAnswers: Record<number, UserAnswer>; // Ответы пользователя в текущей викторине (questionId -> UserAnswer)
  results: UserQuizResult | null; // Финальный результат пользователя
  leaderboard: UserQuizResult[] | null; // Таблица лидеров (полученная по WS или API)
  remainingTime: number | null; // Оставшееся время на текущий вопрос
  quizStatus: 'idle' | 'waiting' | 'active' | 'ended' | 'cancelled'; // Статус викторины с точки зрения пользователя
  isLoading: boolean; // Индикатор загрузки связанных данных
  error: string | null; // Ошибка, связанная с состоянием викторины
}

// Типы для WebSocket Slice (src/store/websocket/types.ts)
export interface WebSocketState {
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null; // Ошибка соединения
  lastMessage: WebSocketMessage | null; // Последнее полученное сообщение (для отладки или общей обработки)
}
Use code with caution.
TypeScript
Примечание: Модели данных обновлены и дополнены на основе newreadme.md. Важно убедиться, что они точно соответствуют API ответам бэкенда. Добавлены комментарии для уточнений.

Клиент HTTP API (Axios)
Реализация Axios клиента с интерцепторами для обработки аутентификации (Cookie+CSRF и Bearer) и обновления токенов, интегрированного с Redux Toolkit.

// src/api/http/client.ts
import axios, { AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { store } from '../../store'; // Путь к вашему Redux store
import { tokenRefreshed, logoutSuccess } from '../../store/auth/slice'; // Экшены из вашего auth slice
import { authService } from '../services/authService'; // Сервис для вызова API обновления токена

let isRefreshing = false;
let failedQueue: Array<{ resolve: (value: unknown) => void; reject: (reason?: any) => void }> = [];

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach(prom => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

// Создаем HTTP клиент с базовыми настройками
export const createApiClient = (config?: AxiosRequestConfig): AxiosInstance => {
  const client = axios.create({
    baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api', // Убедитесь, что URL верный
    timeout: 15000,
    headers: {
      'Content-Type': 'application/json',
      ...config?.headers,
    },
    withCredentials: true, // Важно для cookie-based аутентификации!
    ...config,
  });

  // Интерцептор запросов: Добавляем токен (Bearer или CSRF)
  client.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const { token, csrfToken, useCookieAuth } = store.getState().auth;

      if (useCookieAuth) {
        // Cookie-based Authentication: Добавляем CSRF токен, если он есть
        if (csrfToken) {
          config.headers['X-CSRF-Token'] = csrfToken;
        }
        // Access token не нужен в заголовках, он будет в cookie
        delete config.headers.Authorization;
      } else {
        // Bearer Token Authentication: Добавляем Access Token
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        // CSRF токен не нужен
        delete config.headers['X-CSRF-Token'];
      }
      return config;
    },
    (error) => Promise.reject(error)
  );

  // Интерцептор ответов: Обработка ошибок, особенно 401 для обновления токена
  client.interceptors.response.use(
    (response) => response,
    async (error) => {
      const originalRequest = error.config;
      const status = error.response?.status;
      // Проверяем на 401 и что это не повторный запрос после обновления токена
      if (status === 401 && !originalRequest._retry) {

        // Если уже идет обновление токена, ставим запрос в очередь
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({ resolve, reject });
          })
          .then(token => {
             // Повторяем запрос с новым токеном (если он есть, актуально для Bearer)
             // Для Cookie auth просто повторяем запрос, т.к. cookie обновились
            if (!store.getState().auth.useCookieAuth && token) {
               originalRequest.headers['Authorization'] = 'Bearer ' + token;
            }
            return client(originalRequest);
          })
          .catch(err => {
            return Promise.reject(err); // Пробрасываем ошибку, если очередь обработана с ошибкой
          });
        }

        originalRequest._retry = true; // Помечаем запрос как повторный
        isRefreshing = true;

        try {
          // Вызываем API для обновления токена (через authService)
          // Бэкенд должен использовать refresh-токен из HttpOnly cookie
          const { accessToken, csrfToken: newCsrfToken } = await authService.refreshToken();

          // Обновляем токены в Redux store
          store.dispatch(tokenRefreshed({ token: accessToken, csrfToken: newCsrfToken }));

          // Обновляем заголовок для текущего запроса (актуально для Bearer)
           if (!store.getState().auth.useCookieAuth && accessToken) {
               client.defaults.headers.common['Authorization'] = 'Bearer ' + accessToken;
               originalRequest.headers['Authorization'] = 'Bearer ' + accessToken;
           }
           // Для Cookie auth - обновляем CSRF токен по умолчанию и в текущем запросе
           if (store.getState().auth.useCookieAuth && newCsrfToken) {
               client.defaults.headers.common['X-CSRF-Token'] = newCsrfToken;
               originalRequest.headers['X-CSRF-Token'] = newCsrfToken;
           }

          // Обрабатываем очередь запросов с успехом
          processQueue(null, accessToken);

          // Повторяем исходный запрос с обновленными данными
          return client(originalRequest);
        } catch (refreshError: any) {
          console.error('Unable to refresh token:', refreshError);
          // Обрабатываем очередь запросов с ошибкой
          processQueue(refreshError, null);
          // Если обновить токен не удалось, выполняем выход
          store.dispatch(logoutSuccess()); // Используем экшен из slice
          // Можно перенаправить на страницу логина
          // window.location.href = '/login';
          return Promise.reject(refreshError);
        } finally {
          isRefreshing = false;
        }
      }

      // Пробрасываем другие ошибки
      return Promise.reject(error);
    }
  );

  return client;
};

// Экспортируем синглтон клиента
export const apiClient = createApiClient();
Use code with caution.
TypeScript
Примечание: Добавлен withCredentials: true для работы с cookie. Реализована очередь запросов на время обновления токена. Используется authService для вызова API обновления.

Клиент WebSocket API
Реализация кастомного WebSocket клиента, интегрированного с Redux Toolkit для обновления состояния и получения токена.

// src/api/websocket/client.ts
import { store } from '../../store'; // Путь к вашему Redux store
import {
  wsConnecting, // Добавлен экшен начала подключения
  wsConnected,
  wsDisconnected,
  wsError,
  wsMessage
} from '../../store/websocket/slice'; // Экшены из websocket slice
import { tokenRefreshed, logoutSuccess } from '../../store/auth/slice'; // Экшены из auth slice
import { authService } from '../services/authService'; // Сервис для вызова API обновления токена

// Вспомогательная функция для получения токена (зависит от режима auth)
const getWebSocketToken = (): string | null => {
  const { token, useCookieAuth } = store.getState().auth;
  // Для WebSocket всегда используем Access Token (если есть),
  // т.к. WS не может легко работать с HttpOnly cookie и CSRF.
  // Бэкенд должен поддерживать аутентификацию WS по токену в query параметре.
  // Если используется только Cookie Auth для HTTP, возможно, нужен отдельный механизм для WS Auth.
  // Пока предполагаем, что Access Token доступен всегда, когда пользователь залогинен.
  return token;
};


export class WebSocketClient {
  private static instance: WebSocketClient;
  private socket: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private missedHeartbeats = 0;
  private messageHandlers = new Map<string, Array<(data: any) => void>>();
  private connectionPromise: Promise<void> | null = null;
  private isConnecting = false;

  private constructor() {}

  public static getInstance(): WebSocketClient {
    if (!WebSocketClient.instance) {
      WebSocketClient.instance = new WebSocketClient();
    }
    return WebSocketClient.instance;
  }

  // Подключение к WebSocket серверу
  public connect(): Promise<void> {
    // Если уже подключены, возвращаем разрешенный промис
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      return Promise.resolve();
    }

    // Если уже идет подключение, возвращаем существующий промис
    if (this.isConnecting && this.connectionPromise) {
        return this.connectionPromise;
    }

    this.isConnecting = true;
    store.dispatch(wsConnecting()); // Диспатчим начало подключения

    this.connectionPromise = new Promise(async (resolve, reject) => {
      // Получаем токен для WebSocket
      let wsToken = getWebSocketToken();

      // Если токена нет, но пользователь аутентифицирован (возможно, только cookie),
      // пытаемся получить access token через refresh.
      // Это спорный момент - зависит от логики бэкенда.
      // Возможно, WS должен работать только если есть Bearer/Access token.
      if (!wsToken && store.getState().auth.isAuthenticated) {
          console.warn("No access token for WebSocket, trying to refresh...");
          try {
              const { accessToken } = await authService.refreshToken();
              store.dispatch(tokenRefreshed({ token: accessToken }));
              wsToken = accessToken;
          } catch (refreshError) {
              console.error("Failed to refresh token for WebSocket:", refreshError);
              store.dispatch(logoutSuccess()); // Выход, если не можем получить токен
              this.isConnecting = false;
              reject(new Error('Authentication required for WebSocket'));
              return;
          }
      }

      // Если токена все еще нет (не аутентифицирован или refresh не удался)
      if (!wsToken) {
          console.error('WebSocket connection cancelled: User not authenticated or no token available.');
          this.isConnecting = false;
          reject(new Error('Authentication required'));
          return;
      }

      const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws'}?token=${wsToken}`;
      console.log('Connecting to WebSocket:', wsUrl.replace(wsToken, '***')); // Не логируем токен

      // Закрываем старое соединение, если оно есть
      this.disconnectInternal(true); // true - не пытаться переподключаться

      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = () => {
        console.log('WebSocket connection established');
        this.reconnectAttempts = 0;
        this.clearReconnectTimeout();
        this.startHeartbeat();
        store.dispatch(wsConnected());
        this.isConnecting = false;
        resolve();
      };

      this.socket.onclose = (event) => {
        console.log('WebSocket connection closed', event.code, event.reason);
        this.stopHeartbeat();
        const wasConnected = store.getState().websocket.isConnected; // Проверяем, было ли соединение установлено
        store.dispatch(wsDisconnected({ code: event.code, reason: event.reason }));
        this.isConnecting = false; // Сбрасываем флаг подключения при закрытии

        // Код 4001 используется бэкендом для ошибки аутентификации (невалидный/истекший токен)
        if (event.code === 4001) {
          console.log('WebSocket authentication error (4001). Attempting token refresh...');
          // Пытаемся обновить токен и переподключиться
          this.handleAuthError(reject); // Передаем reject для изначального промиса
        }
        // Код 1000 - нормальное закрытие, не переподключаемся
        // Остальные коды - пытаемся переподключиться, если соединение было установлено
        else if (event.code !== 1000 && wasConnected) {
          this.handleReconnection(reject); // Передаем reject
        } else {
            // Если соединение не было установлено и закрылось не из-за auth error,
            // значит изначальное подключение не удалось.
            if (!wasConnected && event.code !== 4001) {
                 reject(new Error(`WebSocket connection failed: ${event.code} ${event.reason}`));
            }
            // Если было закрыто нормально (1000) или не было установлено, просто завершаем.
        }
      };

      this.socket.onerror = (errorEvent) => {
        // Ошибка может возникнуть ДО onclose, если не удалось подключиться
        console.error('WebSocket error:', errorEvent);
        const errorMsg = 'WebSocket connection error';
        store.dispatch(wsError({ error: errorMsg }));
        this.isConnecting = false; // Сбрасываем флаг
        // Если промис еще не завершен, отклоняем его
        if (this.connectionPromise) {
            reject(new Error(errorMsg));
        }
        // Не вызываем handleReconnection здесь, ждем onclose
      };

      this.socket.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data as string);

          // Обработка heartbeat от сервера
          if (message.type === 'server:heartbeat') {
            this.missedHeartbeats = 0;
            return;
          }

          // Диспатч сообщения в Redux store (для логгирования или общей обработки)
          store.dispatch(wsMessage(message));

          // Вызов зарегистрированных обработчиков для данного типа сообщений
          const handlers = this.messageHandlers.get(message.type) || [];
          if (handlers.length > 0) {
            handlers.forEach(handler => {
              try {
                handler(message.data);
              } catch (handlerError) {
                console.error(`Error in WebSocket message handler for type "${message.type}":`, handlerError);
              }
            });
          } else {
            // console.warn(`No WebSocket message handlers registered for type: ${message.type}`);
          }
        } catch (parseError) {
          console.error('Failed to parse WebSocket message:', parseError, event.data);
          store.dispatch(wsError({ error: 'Failed to parse WebSocket message' }));
        }
      };
    })
    .finally(() => {
        // Очищаем промис после его завершения (успех или ошибка)
        // this.connectionPromise = null; // Не очищаем здесь, чтобы повторные вызовы connect возвращали результат
        this.isConnecting = false; // Убеждаемся, что флаг сброшен
    });

    return this.connectionPromise;
  }

  // Обработка ошибки аутентификации WebSocket (код 4001)
  private async handleAuthError(reject: (reason?: any) => void): Promise<void> {
      try {
          const { accessToken } = await authService.refreshToken();
          store.dispatch(tokenRefreshed({ token: accessToken }));
          console.log('Token refreshed successfully after WS auth error. Reconnecting WebSocket...');
          // Не вызываем connect() напрямую, чтобы избежать зацикливания, если refresh не помогает
          // Вместо этого полагаемся на механизм handleReconnection, который будет вызван из onclose
          this.handleReconnection(reject);
      } catch (refreshError) {
          console.error('Failed to refresh token after WS auth error:', refreshError);
          store.dispatch(logoutSuccess()); // Выход, если не удалось обновить токен
          reject(new Error('WebSocket authentication failed and token refresh failed')); // Отклоняем промис подключения
      }
  }


  // Отправка сообщения через WebSocket
  public sendMessage(type: string, data: any, priority: 'LOW' | 'NORMAL' | 'HIGH' | 'CRITICAL' = 'NORMAL'): boolean {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.warn(`Attempted to send WebSocket message "${type}" but connection is not open (state: ${this.socket?.readyState}).`);
      // Можно добавить логику постановки в очередь, если нужно
      return false;
    }

    const message: WebSocketMessage = { type, data, priority };
    try {
      this.socket.send(JSON.stringify(message));
      // console.log('Sent WS message:', type, data); // Для отладки
      return true;
    } catch (error) {
      console.error(`Failed to send WebSocket message "${type}":`, error);
      store.dispatch(wsError({ error: `Failed to send WebSocket message: ${type}` }));
      return false;
    }
  }

  // Закрытие соединения (публичный метод)
  public disconnect(): void {
      this.disconnectInternal(false); // false - может пытаться переподключиться, если не штатное закрытие
      this.clearReconnectTimeout(); // Явно отменяем таймер переподключения при ручном дисконнекте
      console.log('WebSocket disconnected by client request.');
  }

  // Внутренний метод закрытия соединения
  private disconnectInternal(isPlanned: boolean = false): void {
      if (this.socket) {
          // Удаляем обработчики, чтобы избежать их вызова после закрытия
          this.socket.onopen = null;
          this.socket.onmessage = null;
          this.socket.onerror = null;
          this.socket.onclose = null; // Особенно важно, чтобы избежать рекурсии или лишних reconnect

          if (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING) {
              this.socket.close(isPlanned ? 1000 : 1001, isPlanned ? 'Client disconnected normally' : 'Client initiated disconnect');
          }
          this.socket = null;
      }
      this.stopHeartbeat();
      // Не сбрасываем reconnectAttempts здесь, это делается при успешном onopen
  }

  // Очистка таймера переподключения
  private clearReconnectTimeout(): void {
      if (this.reconnectTimeout) {
          clearTimeout(this.reconnectTimeout);
          this.reconnectTimeout = null;
      }
  }


  // Добавление обработчика для определенного типа сообщений
  public addMessageHandler(type: string, handler: (data: any) => void): () => void {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, []);
    }
    const handlers = this.messageHandlers.get(type)!;
    handlers.push(handler);
    console.log(`Added WS message handler for type: ${type}`);

    // Возвращаем функцию для удаления этого конкретного обработчика
    return () => this.removeMessageHandler(type, handler);
  }

  // Удаление обработчика
  public removeMessageHandler(type: string, handler: (data: any) => void): void {
    const handlers = this.messageHandlers.get(type);
    if (!handlers) {
      return;
    }

    const index = handlers.indexOf(handler);
    if (index !== -1) {
      handlers.splice(index, 1);
      console.log(`Removed WS message handler for type: ${type}`);
      if (handlers.length === 0) {
        this.messageHandlers.delete(type);
      }
    }
  }

  // Запуск проверки соединения (heartbeat)
  private startHeartbeat(): void {
    this.stopHeartbeat(); // Убедимся, что старый интервал остановлен

    this.heartbeatInterval = setInterval(() => {
      if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        if (this.missedHeartbeats >= 2) {
          console.warn(`WebSocket missed ${this.missedHeartbeats} heartbeats. Closing connection and attempting reconnect...`);
          this.socket.close(1001, 'Heartbeat missed'); // Закрываем соединение
          // Переподключение будет обработано в onclose
          this.stopHeartbeat(); // Останавливаем интервал, чтобы избежать повторных попыток
        } else {
          // Отправляем пинг только если соединение открыто
          if (this.sendMessage('user:heartbeat', {})) {
            this.missedHeartbeats++;
            // console.log(`Sent user:heartbeat (missed: ${this.missedHeartbeats})`); // Для отладки
          }
        }
      } else {
         // Если сокет закрыт или не в OPEN состоянии, просто останавливаем heartbeat
         this.stopHeartbeat();
      }
    }, 30000); // Каждые 30 секунд
  }

  // Остановка проверки соединения
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
    this.missedHeartbeats = 0; // Сбрасываем счетчик пропущенных
  }

  // Обработка переподключения
  private handleReconnection(reject?: (reason?: any) => void): void {
    this.clearReconnectTimeout(); // Очищаем предыдущий таймаут, если есть

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('WebSocket maximum reconnect attempts reached. Stopping reconnection.');
      store.dispatch(wsError({ error: 'Maximum reconnect attempts reached' }));
      if (reject) {
          reject(new Error('Maximum reconnect attempts reached')); // Отклоняем изначальный промис подключения
      }
      // Возможно, стоит сделать logout, если WS критичен
      // store.dispatch(logoutSuccess());
      return;
    }

    this.reconnectAttempts++;

    // Экспоненциальная задержка с джиттером
    const baseDelay = 1000 * Math.pow(2, this.reconnectAttempts - 1);
    const jitter = baseDelay * 0.2 * Math.random(); // +/- 10% jitter
    const delay = Math.min(baseDelay + jitter, 30000); // Макс. задержка 30 секунд

    console.log(`WebSocket attempting to reconnect in ${Math.round(delay / 1000)}s (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

    this.reconnectTimeout = setTimeout(() => {
      // Не передаем reject дальше, т.к. это уже попытка переподключения
      this.connect().catch(connectError => {
          console.error(`WebSocket reconnect attempt ${this.reconnectAttempts} failed:`, connectError.message);
          // Ошибка будет обработана в onclose/onerror следующего вызова connect
      });
    }, delay);
  }
}

// Создаем и экспортируем синглтон клиента
export const wsClient = WebSocketClient.getInstance();
Use code with caution.
TypeScript
Примечание: WS клиент адаптирован для получения токена из Redux store. Добавлена обработка кода 4001 для ошибки аутентификации WS с попыткой обновления токена. Добавлен экшен wsConnecting. Улучшена логика переподключения и обработки ошибок.

Управление состоянием (Redux Toolkit)
Подход к организации хранилища состояния с использованием Redux Toolkit, разделенного на слайсы.

// src/store/index.ts
import { configureStore } from '@reduxjs/toolkit';
import authReducer from './auth/slice';
import quizReducer from './quiz/slice';
import websocketReducer from './websocket/slice';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    quiz: quizReducer,
    websocket: websocketReducer,
  },
  // Middleware можно добавить здесь (например, для React Query logger)
});

// Типы для использования с React-Redux хуками
export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

// Пример кастомных хуков для удобства (в src/hooks/redux-hooks.ts)
/*
import { TypedUseSelectorHook, useDispatch, useSelector } from 'react-redux';
import type { RootState, AppDispatch } from '../store';

export const useAppDispatch = () => useDispatch<AppDispatch>();
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector;
*/
Use code with caution.
TypeScript
Слайс для аутентификации (src/store/auth/slice.ts)
Управляет состоянием пользователя, токенами (access и CSRF) и флагом режима аутентификации.

// src/store/auth/slice.ts
import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { User } from '../../types/user'; // Импорт типа User
import { AuthState } from './types'; // Импорт типа AuthState

// Начальное состояние, можно попытаться загрузить из localStorage, если нужно
const initialState: AuthState = {
  user: null,
  token: null, // Access Token (для Bearer)
  csrfToken: null, // CSRF Token (для Cookie Auth)
  isAuthenticated: false,
  isLoading: false, // Индикатор загрузки при логине/регистрации/refresh
  error: null,
  useCookieAuth: true, // По умолчанию используем Cookie Auth как более безопасный
};

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    // Экшен при начале запроса на логин/регистрацию
    authRequestStart: (state) => {
      state.isLoading = true;
      state.error = null;
    },
    // Экшен при успешном логине/регистрации/получении пользователя
    authSuccess: (state, action: PayloadAction<{ user: User; token?: string; csrfToken?: string }>) => {
      state.isLoading = false;
      state.isAuthenticated = true;
      state.user = action.payload.user;
      state.token = action.payload.token || null; // Сохраняем access token, если он пришел (для Bearer)
      state.csrfToken = action.payload.csrfToken || null; // Сохраняем CSRF токен, если он пришел (для Cookie)
      state.error = null;
    },
    // Экшен при ошибке логина/регистрации
    authFailure: (state, action: PayloadAction<string>) => {
      state.isLoading = false;
      state.isAuthenticated = false;
      state.user = null;
      state.token = null;
      state.csrfToken = null;
      state.error = action.payload;
    },
    // Экшен при успешном выходе
    logoutSuccess: (state) => {
      state.user = null;
      state.token = null;
      state.csrfToken = null;
      state.isAuthenticated = false;
      state.isLoading = false;
      state.error = null;
      // Можно сбросить и другие состояния здесь, если нужно
    },
    // Экшен при успешном обновлении токенов
    tokenRefreshed: (state, action: PayloadAction<{ token?: string; csrfToken?: string }>) => {
      // Обновляем только те токены, которые пришли в payload
      if (action.payload.token !== undefined) {
          state.token = action.payload.token;
      }
      if (action.payload.csrfToken !== undefined) {
          state.csrfToken = action.payload.csrfToken;
      }
      state.isLoading = false; // Завершаем индикатор загрузки, если он был из-за refresh
      state.isAuthenticated = true; // Подтверждаем аутентификацию
    },
    // Экшен для обновления данных пользователя (например, после редактирования профиля)
    updateUser: (state, action: PayloadAction<Partial<User>>) => {
      if (state.user) {
        state.user = { ...state.user, ...action.payload };
      }
    },
    // Экшен для переключения режима аутентификации (если это нужно делать динамически)
    setAuthMode: (state, action: PayloadAction<'cookie' | 'bearer'>) => {
      state.useCookieAuth = action.payload === 'cookie';
      // При смене режима стоит сбросить токены другого режима
      if (state.useCookieAuth) {
          state.token = null;
      } else {
          state.csrfToken = null;
      }
    },
    // Очистка ошибки аутентификации
    clearAuthError: (state) => {
        state.error = null;
    }
  },
});

export const {
  authRequestStart,
  authSuccess,
  authFailure,
  logoutSuccess,
  tokenRefreshed,
  updateUser,
  setAuthMode,
  clearAuthError,
} = authSlice.actions;

export default authSlice.reducer;
Use code with caution.
TypeScript
Слайс для WebSocket (src/store/websocket/slice.ts)
Управляет состоянием WebSocket соединения.

// src/store/websocket/slice.ts
import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { WebSocketState, WebSocketMessage } from './types'; // Импорт типов

const initialState: WebSocketState = {
  isConnected: false,
  isConnecting: false, // Добавлен флаг процесса подключения
  error: null,
  lastMessage: null,
};

const websocketSlice = createSlice({
  name: 'websocket',
  initialState,
  reducers: {
    // Экшен при начале подключения
    wsConnecting: (state) => {
      state.isConnecting = true;
      state.isConnected = false; // Убеждаемся, что не считаемся подключенными
      state.error = null;
    },
    // Экшен при успешном подключении
    wsConnected: (state) => {
      state.isConnected = true;
      state.isConnecting = false;
      state.error = null;
    },
    // Экшен при закрытии соединения
    wsDisconnected: (state, action: PayloadAction<{ code: number; reason: string }>) => {
      state.isConnected = false;
      state.isConnecting = false;
      // Записываем ошибку, только если закрытие не было штатным (код 1000)
      // и если не было ошибки аутентификации (код 4001), т.к. она обрабатывается отдельно
      if (action.payload.code !== 1000 && action.payload.code !== 4001) {
        state.error = `WebSocket closed: ${action.payload.code} ${action.payload.reason || 'No reason specified'}`;
      } else if (action.payload.code === 1000) {
          state.error = null; // Очищаем ошибку при штатном закрытии
      }
      // При ошибке 4001 не меняем error здесь, ждем результат refresh
    },
    // Экшен при ошибке WebSocket (например, ошибка сети, парсинга)
    wsError: (state, action: PayloadAction<{ error: string }>) => {
      // Не устанавливаем isConnecting = false здесь, т.к. ошибка может быть временной
      state.error = action.payload.error;
      // Можно добавить логику, чтобы сбросить isConnected, если ошибка фатальна
      // state.isConnected = false;
    },
    // Экшен при получении нового сообщения
    wsMessage: (state, action: PayloadAction<WebSocketMessage>) => {
      state.lastMessage = action.payload;
      // Основная обработка сообщения должна быть в компонентах или thunks/sagas,
      // которые подписываются на WS Client или реагируют на этот экшен.
    },
    // Экшен для очистки ошибки
    clearWSError: (state) => {
      state.error = null;
    },
  },
});

export const {
  wsConnecting,
  wsConnected,
  wsDisconnected,
  wsError,
  wsMessage,
  clearWSError,
} = websocketSlice.actions;

export default websocketSlice.reducer;
Use code with caution.
TypeScript
Слайс для викторин (src/store/quiz/slice.ts)
Управляет состоянием активной викторины, которое обновляется преимущественно через WebSocket. Данные о списке викторин или деталях неактивных викторин лучше хранить через React Query.

// src/store/quiz/slice.ts
import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Quiz } from '../../types/quiz';
import { Question } from '../../types/question';
import { UserAnswer } from '../../types/answer';
import { UserQuizResult } from '../../types/result';
import { QuizState } from './types'; // Импорт типа QuizState

const initialState: QuizState = {
  activeQuiz: null,
  currentQuestion: null,
  userAnswers: {},
  results: null,
  leaderboard: null,
  remainingTime: null,
  quizStatus: 'idle',
  isLoading: false, // Может использоваться при инициализации викторины
  error: null,
};

const quizSlice = createSlice({
  name: 'quiz',
  initialState,
  reducers: {
    // Установка активной викторины (обычно при получении события QUIZ_START по WS)
    setActiveQuiz: (state, action: PayloadAction<{ quiz: Quiz; questions?: Question[] }>) => {
      state.activeQuiz = action.payload.quiz;
      state.questions = action.payload.questions || []; // Если вопросы приходят сразу
      state.quizStatus = 'waiting'; // Или 'active', если викторина начинается сразу
      state.currentQuestion = null;
      state.userAnswers = {};
      state.results = null;
      state.leaderboard = null;
      state.error = null;
      state.isLoading = false;
    },
    // Установка текущего вопроса (при получении QUESTION_START по WS)
    setCurrentQuestion: (state, action: PayloadAction<Question>) => {
      // Добавляем вопрос в список, если его там еще нет
      if (!state.questions.some(q => q.id === action.payload.id)) {
          state.questions.push(action.payload);
      }
      state.currentQuestion = action.payload;
      state.quizStatus = 'active';
      state.remainingTime = action.payload.time_limit_sec;
      state.error = null;
    },
    // Добавление/обновление ответа пользователя (после отправки через WS и получения подтверждения/результата)
    // Или можно обновлять оптимистично при отправке
    updateUserAnswer: (state, action: PayloadAction<UserAnswer>) => {
      state.userAnswers[action.payload.question_id] = action.payload;
      // Можно добавить логику расчета промежуточного счета
    },
    // Обновление таймера вопроса (можно делать локально или по WS)
    updateRemainingTime: (state, action: PayloadAction<number>) => {
      if (state.quizStatus === 'active') {
        state.remainingTime = Math.max(0, action.payload);
      }
    },
    // Завершение вопроса (при получении QUESTION_END по WS)
    endCurrentQuestion: (state, action: PayloadAction<{ questionId: number; correctAnswer?: number; /* другие данные */ }>) => {
        if (state.currentQuestion?.id === action.payload.questionId) {
            state.currentQuestion = null; // Или пометить как завершенный
            state.remainingTime = 0;
            // Можно здесь показать правильный ответ, если он пришел
        }
    },
    // Завершение викторины (при получении QUIZ_END по WS)
    setQuizEnded: (state) => {
      state.quizStatus = 'ended';
      state.currentQuestion = null;
      state.remainingTime = null;
      // Результаты и лидерборд обычно приходят отдельными сообщениями
    },
    // Отмена викторины
    setQuizCancelled: (state) => {
      state.quizStatus = 'cancelled';
      state.activeQuiz = null;
      state.currentQuestion = null;
      state.remainingTime = null;
      state.userAnswers = {};
    },
    // Установка финальных результатов пользователя (при получении RESULT_UPDATE по WS)
    setUserResults: (state, action: PayloadAction<UserQuizResult>) => {
      state.results = action.payload;
    },
    // Установка таблицы лидеров (при получении LEADERBOARD_UPDATE по WS)
    setLeaderboard: (state, action: PayloadAction<UserQuizResult[]>) => {
      state.leaderboard = action.payload;
    },
    // Сброс состояния викторины (при выходе пользователя или завершении)
    resetQuizState: (state) => {
      return initialState; // Возвращаем начальное состояние
    },
     // Установка ошибки, специфичной для викторины
    setQuizError: (state, action: PayloadAction<string>) => {
        state.error = action.payload;
        state.isLoading = false;
    },
    // Начало загрузки данных викторины
    quizLoadingStart: (state) => {
        state.isLoading = true;
        state.error = null;
    }
  },
});

export const {
  setActiveQuiz,
  setCurrentQuestion,
  updateUserAnswer,
  updateRemainingTime,
  endCurrentQuestion,
  setQuizEnded,
  setQuizCancelled,
  setUserResults,
  setLeaderboard,
  resetQuizState,
  setQuizError,
  quizLoadingStart
} = quizSlice.actions;

export default quizSlice.reducer;
Use code with caution.
TypeScript
Кэширование данных (React Query)
Использование React Query для получения, кэширования и инвалидации данных из REST API. Хуки размещаются в src/hooks/query-hooks/.

// src/hooks/query-hooks/useQuizQueries.ts
import { useQuery, useMutation, useQueryClient, UseQueryOptions } from '@tanstack/react-query';
import { quizService } from '../../api/services/quizService'; // Путь к вашему сервису
import { Quiz, Question, UserQuizResult } from '../../types'; // Импорт типов

// Ключи запросов (для инвалидации и управления кэшем)
const quizKeys = {
  all: ['quizzes'] as const,
  lists: (filters?: any) => [...quizKeys.all, 'list', filters] as const,
  details: (id: number) => [...quizKeys.all, 'detail', id] as const,
  active: () => [...quizKeys.all, 'active'] as const,
  scheduled: () => [...quizKeys.all, 'scheduled'] as const,
  results: (id: number) => [...quizKeys.details(id), 'results'] as const,
  userResult: (quizId: number, userId?: number) => [...quizKeys.results(quizId), 'user', userId ?? 'current'] as const,
  questions: (id: number) => [...quizKeys.details(id), 'questions'] as const,
};

// --- Queries ---

// Получение списка викторин с пагинацией
export const useQuizzes = (page = 1, pageSize = 10, filters?: Record<string, any>) => {
  return useQuery({
    queryKey: quizKeys.lists({ page, pageSize, ...filters }),
    queryFn: () => quizService.getQuizzes(page, pageSize, filters),
    staleTime: 60 * 1000, // 1 минута
    keepPreviousData: true, // Полезно для пагинации
  });
};

// Получение активной викторины
export const useActiveQuiz = (options?: UseQueryOptions<Quiz | null, Error>) => {
  return useQuery({
    queryKey: quizKeys.active(),
    queryFn: () => quizService.getActiveQuiz(), // Предполагаем, что есть такой метод в сервисе
    staleTime: 15 * 1000, // 15 секунд (чаще обновляем, т.к. статус может измениться)
    refetchInterval: 30 * 1000, // Перепроверка каждые 30 секунд
    ...options,
  });
};

// Получение запланированных викторин
export const useScheduledQuizzes = (options?: UseQueryOptions<Quiz[], Error>) => {
  return useQuery({
    queryKey: quizKeys.scheduled(),
    queryFn: () => quizService.getScheduledQuizzes(), // Предполагаем, что есть такой метод
    staleTime: 5 * 60 * 1000, // 5 минут
    ...options,
  });
};

// Получение деталей викторины по ID
export const useQuiz = (id: number | undefined, options?: UseQueryOptions<Quiz, Error>) => {
  return useQuery({
    queryKey: quizKeys.details(id!),
    queryFn: () => quizService.getQuiz(id!), // Метод для получения базовой информации
    staleTime: 5 * 60 * 1000, // 5 минут
    enabled: !!id, // Запрос выполняется только если id предоставлен
    ...options,
  });
};

// Получение викторины с вопросами по ID
export const useQuizWithQuestions = (id: number | undefined, options?: UseQueryOptions<Quiz & { questions: Question[] }, Error>) => {
  return useQuery({
    queryKey: quizKeys.questions(id!),
    queryFn: () => quizService.getQuizWithQuestions(id!), // Метод API: /api/quizzes/:id/with-questions
    staleTime: 10 * 60 * 1000, // Кэшируем дольше, т.к. вопросы редко меняются после создания
    enabled: !!id,
    ...options,
  });
};


// Получение результатов (лидерборда) викторины по ID
export const useQuizResults = (id: number | undefined, options?: UseQueryOptions<UserQuizResult[], Error>) => {
  return useQuery({
    queryKey: quizKeys.results(id!),
    queryFn: () => quizService.getQuizResults(id!), // Метод API: /api/quizzes/:id/results
    staleTime: 60 * 1000, // 1 минута
    enabled: !!id,
    ...options,
  });
};

// Получение персонального результата пользователя для викторины
export const useUserQuizResult = (id: number | undefined, options?: UseQueryOptions<UserQuizResult, Error>) => {
  return useQuery({
    queryKey: quizKeys.userResult(id!), // Ключ без userId - для текущего пользователя
    queryFn: () => quizService.getUserQuizResult(id!), // Метод API: /api/quizzes/:id/my-result
    staleTime: 60 * 1000, // 1 минута
    enabled: !!id,
    ...options,
  });
};


// --- Mutations ---

export const useCreateQuiz = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (quizData: Partial<Quiz>) => quizService.createQuiz(quizData), // API: POST /api/quizzes
    onSuccess: (newQuiz) => {
      // Инвалидируем списки, чтобы они обновились
      queryClient.invalidateQueries({ queryKey: quizKeys.lists() });
      queryClient.invalidateQueries({ queryKey: quizKeys.scheduled() });
      // Можно сразу добавить новую викторину в кэш, если нужно
      // queryClient.setQueryData(quizKeys.details(newQuiz.id), newQuiz);
    },
    // onError: (error) => { ... }
  });
};

export const useAddQuestions = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ quizId, questions }: { quizId: number; questions: Partial<Question>[] }) =>
      quizService.addQuestions(quizId, questions), // API: POST /api/quizzes/:id/questions
    onSuccess: (_, variables) => {
      // Инвалидируем кэш вопросов и деталей для этой викторины
      queryClient.invalidateQueries({ queryKey: quizKeys.questions(variables.quizId) });
      queryClient.invalidateQueries({ queryKey: quizKeys.details(variables.quizId) });
      // Инвалидируем списки, т.к. могло измениться question_count
      queryClient.invalidateQueries({ queryKey: quizKeys.lists() });
    },
  });
};

export const useScheduleQuiz = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ quizId, time }: { quizId: number; time: string }) =>
      quizService.scheduleQuiz(quizId, time), // API: PUT /api/quizzes/:id/schedule
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: quizKeys.details(variables.quizId) });
      queryClient.invalidateQueries({ queryKey: quizKeys.scheduled() });
      queryClient.invalidateQueries({ queryKey: quizKeys.lists() }); // Статус изменился
    },
  });
};

export const useCancelQuiz = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (quizId: number) => quizService.cancelQuiz(quizId), // API: PUT /api/quizzes/:id/cancel
    onSuccess: (_, quizId) => {
      queryClient.invalidateQueries({ queryKey: quizKeys.details(quizId) });
      queryClient.invalidateQueries({ queryKey: quizKeys.scheduled() });
      queryClient.invalidateQueries({ queryKey: quizKeys.lists() }); // Статус изменился
    },
  });
};

// Добавить хуки для User (useCurrentUser, useUpdateProfile и т.д. по аналогии)
// src/hooks/query-hooks/useUserQueries.ts
Use code with caution.
TypeScript
Примечание: Используется @tanstack/react-query (v4+). Добавлены Query Keys для лучшего управления кэшем. Добавлены мутации для действий с викторинами.

Инициализация WebSocket
Инициализация WebSocket клиента при запуске приложения в _app.tsx с использованием React хуков и Redux.

// src/pages/_app.tsx
import { useEffect } from 'react';
import { AppProps } from 'next/app';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider, Hydrate } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { store } from '../store';
import { wsClient } from '../api/websocket/client';
import { useAppSelector } from '../hooks/redux-hooks'; // Кастомный хук useSelector
import ErrorBoundary from '../components/ErrorBoundary'; // Ваш ErrorBoundary
import '../styles/globals.css'; // Пример глобальных стилей
import { Toaster } from 'react-hot-toast'; // Пример библиотеки для уведомлений

// Создаем QueryClient один раз
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1, // Повторять запросы 1 раз при ошибке
      refetchOnWindowFocus: true, // Обновлять данные при фокусе на окне
    },
  },
});

// Компонент для управления WebSocket соединением
const WebSocketManager = () => {
  // Получаем статус аутентификации и флаг подключения WS из Redux
  const isAuthenticated = useAppSelector(state => state.auth.isAuthenticated);
  const isConnected = useAppSelector(state => state.websocket.isConnected);
  const isConnecting = useAppSelector(state => state.websocket.isConnecting);

  useEffect(() => {
    let unsubscribeHandlers: (() => void)[] = [];

    if (isAuthenticated && !isConnected && !isConnecting) {
      console.log('User authenticated, attempting to connect WebSocket...');
      // Подключаемся
      wsClient.connect()
          .then(() => {
              console.log('WebSocket connected successfully via Manager.');
              // --- Регистрируем обработчики WS сообщений здесь ---
              // Пример:
              // const unsubQuizStart = wsClient.addMessageHandler('QUIZ_START', handleQuizStart);
              // const unsubQuestionStart = wsClient.addMessageHandler('QUESTION_START', handleQuestionStart);
              // ...
              // unsubscribeHandlers = [unsubQuizStart, unsubQuestionStart, ...];

              // !!! Важно: Функции обработчики (handleQuizStart и т.д.) должны быть определены
              // вне useEffect или импортированы. Они могут диспатчить экшены в Redux store.
              // Например: const handleQuizStart = (data) => store.dispatch(setActiveQuiz(data));
          })
          .catch(error => {
              console.error('WebSocket initial connection failed:', error.message);
              // Ошибка уже должна быть в Redux store через wsError
          });

    } else if (!isAuthenticated && isConnected) {
      // Если пользователь вышел, а соединение еще активно, отключаемся
      console.log('User logged out, disconnecting WebSocket...');
      wsClient.disconnect();
    }

    // Функция очистки при размонтировании компонента или изменении isAuthenticated
    return () => {
      // Отписываемся от всех обработчиков сообщений
      unsubscribeHandlers.forEach(unsubscribe => unsubscribe());
      unsubscribeHandlers = [];

      // Не отключаем WS здесь при каждом ререндере,
      // отключаем только если пользователь разлогинился (см. условие выше)
      // или если компонент _app полностью размонтируется (маловероятно)
      // wsClient.disconnect();
    };
    // Зависимости: isAuthenticated - для подключения/отключения,
    // isConnected, isConnecting - чтобы не пытаться подключиться повторно, если уже идет процесс или подключено
  }, [isAuthenticated, isConnected, isConnecting]);

  // Этот компонент ничего не рендерит
  return null;
};

function MyApp({ Component, pageProps }: AppProps<{ dehydratedState: unknown }>) {
  return (
    // Оборачиваем в ErrorBoundary для глобальной обработки ошибок рендеринга
    <ErrorBoundary fallback={<div>Something went wrong</div>}>
      <Provider store={store}>
        <QueryClientProvider client={queryClient}>
          {/* Hydrate используется для SSR/SSG с React Query */}
          <Hydrate state={pageProps.dehydratedState}>
            {/* Компонент для управления WS */}
            <WebSocketManager />
            {/* Основной компонент страницы */}
            <Component {...pageProps} />
            {/* Место для глобальных компонентов, например, уведомлений */}
            <Toaster position="bottom-right" />
          </Hydrate>
          {/* Инструменты разработчика React Query */}
          <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
      </Provider>
    </ErrorBoundary>
  );
}

export default MyApp;
Use code with caution.
TypeScript
Примечание: Используется @tanstack/react-query v4+ синтаксис для QueryClientProvider и Hydrate. Добавлен ReactQueryDevtools. Логика WebSocketManager улучшена для обработки состояния подключения и регистрации/отписки обработчиков.

Рекомендации по оптимизации
1. Кэширование данных (React Query)
React Query автоматически кэширует данные. Настройте staleTime и cacheTime для оптимизации.

// src/hooks/query-hooks/useCategoryQueries.ts
import { useQuery } from '@tanstack/react-query';
import { categoryService } from '../../api/services/categoryService'; // Пример сервиса

const categoryKeys = {
  all: ['categories'] as const,
  list: () => [...categoryKeys.all, 'list'] as const,
}

// Пример: Кэширование категорий на 24 часа
export function useCategories() {
  return useQuery({
    queryKey: categoryKeys.list(),
    queryFn: () => categoryService.getCategories(),
    staleTime: 24 * 60 * 60 * 1000, // 24 часа - данные считаются свежими
    cacheTime: 24 * 60 * 60 * 1000, // 24 часа - данные удаляются из кэша после неактивности
  });
}
Use code with caution.
TypeScript
2. Оптимизация загрузки (Next.js)
Используйте next/dynamic для ленивой загрузки компонентов, которые не нужны для первого рендера.

// src/pages/quiz/[id].tsx
import dynamic from 'next/dynamic';
import { useRouter } from 'next/router';
import { Spinner } from '../../components/common/Spinner'; // Пример компонента спиннера

// Ленивая загрузка тяжелого компонента деталей викторины
const QuizDetails = dynamic(
  () => import('../../components/quiz/QuizDetails'), // Путь к компоненту
  {
    loading: () => <div className="flex justify-center items-center h-64"><Spinner size="large" /></div>,
    ssr: false // Отключаем SSR для этого компонента, если он чисто клиентский
  }
);

export default function QuizPage() {
  const router = useRouter();
  const { id } = router.query;

  // Конвертируем id в число или показываем загрузку/ошибку
  const quizId = typeof id === 'string' ? parseInt(id, 10) : undefined;

  if (!quizId) {
    // Можно показать спиннер или сообщение об ошибке
    return <div className="flex justify-center items-center h-screen">Loading...</div>;
  }

  return (
      // Обертка макета страницы
      // <MainLayout>
          <QuizDetails quizId={quizId} />
      // </MainLayout>
  );
}
Use code with caution.
TypeScript
3. Оптимизация рендеринга (React)
Используйте React.memo для функциональных компонентов и useCallback, useMemo для мемоизации функций и значений, чтобы предотвратить лишние ререндеры дочерних компонентов.

// src/components/quiz/LeaderboardRow.tsx
import React, { memo } from 'react';
import { UserQuizResult } from '../../types'; // Тип результата

interface LeaderboardRowProps {
  result: UserQuizResult;
  isCurrentUser: boolean;
}

// Компонент строки таблицы лидеров
const LeaderboardRow: React.FC<LeaderboardRowProps> = ({
  result, isCurrentUser
}) => {
  // console.log(`Rendering row for ${result.username}`); // Для отладки

  return (
    <tr className={`border-b ${isCurrentUser ? 'bg-yellow-100 font-bold' : 'hover:bg-gray-50'}`}>
      <td className="px-4 py-2 text-center">{result.rank}</td>
      <td className="px-4 py-2 flex items-center">
        {result.profile_picture && (
            <img src={result.profile_picture} alt={result.username} className="w-8 h-8 rounded-full mr-3 object-cover"/>
        )}
        <span>{result.username}</span>
        </td>
      <td className="px-4 py-2 text-center">{result.score}</td>
      <td className="px-4 py-2 text-center">{result.correct_answers}/{result.total_questions}</td>
    </tr>
  );
};

// Используем React.memo для мемоизации.
// Перерендер произойдет только если изменятся result или isCurrentUser.
// Для сложных объектов в props может потребоваться кастомная функция сравнения.
export default memo(LeaderboardRow);

// Пример использования в компоненте Leaderboard
/*
const Leaderboard = ({ results, currentUserId }) => {
    const sortedResults = useMemo(() => [...results].sort((a, b) => a.rank - b.rank), [results]);

    return (
        <table>
            <thead>...</thead>
            <tbody>
                {sortedResults.map(result => (
                    <LeaderboardRow
                        key={result.user_id}
                        result={result}
                        isCurrentUser={result.user_id === currentUserId}
                    />
                ))}
            </tbody>
        </table>
    );
}
*/
Use code with caution.
TypeScript
4. Оптимизация WebSocket
Дедупликация сообщений: Если есть риск получения дубликатов от сервера (например, при переподключении), проверяйте уникальные идентификаторы сообщений или данных (ID вопроса, ID события) перед обработкой в Redux или компонентах. Пример с lastProcessedQuestionId в вашем исходном документе подходит.

Пакетная обработка: Если сервер может присылать множество мелких обновлений (например, обновление очков каждого игрока), а UI обновляется слишком часто, можно использовать дебаунсинг (lodash/debounce) или троттлинг (lodash/throttle) для экшенов Redux или обновлений состояния компонентов.

Осмысленная подписка/отписка: Управляйте подписками на типы сообщений WS (addMessageHandler/removeMessageHandler) в useEffect компонентов, которые действительно нуждаются в этих данных, чтобы избежать лишней обработки в неактивных частях приложения.

Обработка ошибок
Глобальный обработчик ошибок (React Error Boundary)
Используйте Error Boundary для отлова ошибок рендеринга в дереве компонентов.

// src/components/ErrorBoundary.tsx
import React, { Component, ErrorInfo, ReactNode } from 'react';
import Link from 'next/link';

interface Props {
  children: ReactNode;
  fallback?: ReactNode; // Компонент для отображения при ошибке
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
    errorInfo: null,
  };

  // Статический метод для обновления состояния при ошибке
  public static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  // Метод для логирования информации об ошибке
  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    this.setState({ errorInfo });
    // Здесь можно отправить ошибку в сервис мониторинга (Sentry, LogRocket и т.п.)
    // reportErrorToMonitoringService(error, errorInfo);
  }

  // Функция для сброса состояния ошибки (например, по кнопке)
  private resetError = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
    // Можно добавить логику перезагрузки страницы или перехода
  };

  public render() {
    if (this.state.hasError) {
      // Возвращаем кастомный fallback UI или дефолтный
      return this.props.fallback || (
        <div className="flex flex-col items-center justify-center min-h-screen bg-red-50 text-red-800 p-4">
          <h2 className="text-2xl font-bold mb-4">Что-то пошло не так...</h2>
          <p className="mb-4">Произошла ошибка при отображении этой части страницы.</p>
          {/* Можно показать детали ошибки в режиме разработки */}
          {process.env.NODE_ENV === 'development' && this.state.error && (
            <details className="mb-4 p-4 bg-red-100 rounded w-full max-w-2xl text-sm overflow-auto">
              <summary className="cursor-pointer font-medium">Детали ошибки</summary>
              <pre className="mt-2 whitespace-pre-wrap">
                {this.state.error.toString()}
                {this.state.errorInfo?.componentStack}
              </pre>
            </details>
          )}
          <div className="flex gap-4">
              <button
                onClick={this.resetError}
                className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50"
              >
                Попробовать снова
              </button>
              <Link href="/" legacyBehavior>
                  <a className="px-4 py-2 bg-gray-200 text-gray-800 rounded hover:bg-gray-300">
                      На главную
                  </a>
              </Link>
          </div>
        </div>
      );
    }

    // Если ошибки нет, рендерим дочерние компоненты
    return this.props.children;
  }
}

export default ErrorBoundary;
Use code with caution.
TypeScript
Примечание: Использован TailwindCSS для стилизации. Добавлен вывод деталей ошибки в режиме разработки.

Обработка ошибок API (React Query и Axios)
React Query: Используйте свойства error, isError из хуков useQuery, useMutation для отображения ошибок пользователю.

Axios Interceptor: Интерцептор ответа уже обрабатывает ошибку 401 Unauthorized для обновления токена. Другие ошибки (400, 403, 404, 5xx) будут проброшены дальше и пойманы в React Query.

Компонент для отображения ошибок: Создайте компонент, который принимает объект ошибки (из React Query) и отображает пользователю понятное сообщение.

// src/components/common/ApiErrorAlert.tsx
import React from 'react';
import { AxiosError } from 'axios'; // Тип ошибки Axios

interface ApiErrorAlertProps {
  error: unknown | null; // Ошибка может быть любого типа
  onRetry?: () => void; // Функция для кнопки "Попробовать снова"
  context?: string; // Дополнительный контекст (где произошла ошибка)
}

// Вспомогательная функция для извлечения сообщения
const getApiErrorMessage = (error: unknown): string => {
  if (error instanceof AxiosError) {
    // Ошибка Axios
    const responseData = error.response?.data;
    if (responseData && typeof responseData === 'object') {
        // Ищем поле 'error' или 'message' в ответе API
        if ('error' in responseData && typeof responseData.error === 'string') return responseData.error;
        if ('message' in responseData && typeof responseData.message === 'string') return responseData.message;
    }
    // Если нет специфичного сообщения от API, возвращаем стандартное сообщение Axios
    return error.message || 'Ошибка сети или сервера';
  } else if (error instanceof Error) {
    // Обычная ошибка JavaScript
    return error.message;
  } else if (typeof error === 'string') {
    // Если передали просто строку
    return error;
  }
  // Неизвестный тип ошибки
  return 'Произошла неизвестная ошибка';
};

// Вспомогательная функция для извлечения типа ошибки API
const getApiErrorType = (error: unknown): string | null => {
    if (error instanceof AxiosError) {
        const responseData = error.response?.data;
        if (responseData && typeof responseData === 'object' && 'error_type' in responseData && typeof responseData.error_type === 'string') {
            return responseData.error_type;
        }
        // Можно вернуть статус код как тип
        // return error.response?.status?.toString() || null;
    }
    return null;
}

const ApiErrorAlert: React.FC<ApiErrorAlertProps> = ({ error, onRetry, context }) => {
  if (!error) return null;

  const errorMessage = getApiErrorMessage(error);
  const errorType = getApiErrorType(error); // Получаем error_type из бэкенда, если есть

  // Функция для рендера сообщения в зависимости от типа (можно расширить)
  const renderUserFriendlyMessage = () => {
    switch (errorType) {
      case 'token_expired': // Этот тип обрабатывается интерцептором, но может прийти, если refresh не удался
        return 'Ваша сессия истекла. Пожалуйста, войдите снова.';
      case 'validation_error':
        return `Ошибка валидации${context ? ` в ${context}` : ''}. Проверьте введенные данные. (${errorMessage})`;
      case 'unauthorized':
      case 'forbidden':
           return `Доступ запрещен${context ? ` к ${context}` : ''}. У вас недостаточно прав или требуется вход.`;
      case 'not_found':
           return `Ресурс ${context ? context : ''} не найден.`;
      // Добавить другие типы ошибок с бэкенда
      default:
        // Общее сообщение
        return `Ошибка${context ? ` при ${context}` : ''}: ${errorMessage}`;
    }
  };

  return (
    <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative mb-4" role="alert">
      <strong className="font-bold">Ошибка!</strong>
      <span className="block sm:inline ml-2">{renderUserFriendlyMessage()}</span>
      {onRetry && (
        <button
          onClick={onRetry}
          className="absolute top-0 bottom-0 right-0 px-4 py-3 text-red-700 hover:text-red-900"
          aria-label="Попробовать снова"
        >
          <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9M4 12a9 9 0 1016 5.093V13h-.582m-15.356-2A8.001 8.001 0 0019.418 15M20 12a9 9 0 10-16-5.093V8" />
          </svg>
        </button>
      )}
    </div>
  );
};

export default ApiErrorAlert;

// Пример использования в компоненте
/*
import { useCurrentUser } from '../hooks/query-hooks/useUserQueries';
import ApiErrorAlert from './ApiErrorAlert';
import Spinner from './Spinner';

const UserProfile = () => {
    const { data: user, isLoading, error, refetch } = useCurrentUser();

    if (isLoading) return <Spinner />;

    // Передаем ошибку и функцию refetch для повторной попытки
    if (error) return <ApiErrorAlert error={error} onRetry={refetch} context="загрузке профиля" />;

    return (
        <div>
            <h1>{user?.username}</h1>
            <p>{user?.email}</p>
        </div>
    );
}
*/
Use code with caution.
TypeScript
Заключение
Для разработки фронтенд-части Trivia API настоятельно рекомендуется использовать стек React, Next.js и TypeScript. Этот выбор обеспечивает:

Надежность и Масштабируемость: TypeScript и четкая структура проекта.

Производительность: Оптимизации Next.js (SSR/SSG/ISR, code splitting), React Query для кэширования и React.memo/useCallback для оптимизации рендеринга.

Современный Developer Experience: Быстрая разработка с TailwindCSS, готовые решения для состояния (React Query, Redux Toolkit), удобный роутинг Next.js.

Совместимость с Бэкендом:

Аутентификация: Next.js и Axios с интерцепторами хорошо подходят для обработки как Cookie+CSRF, так и Bearer токенов, поддерживаемых вашим Go бэкендом.

Real-time: Надежный кастомный WebSocket клиент интегрируется с Redux для управления состоянием викторины в реальном времени.

API Взаимодействие: React Query элегантно управляет взаимодействием с REST API.

Следуя этим рекомендациям, вы сможете создать надежное, производительное и удобное в поддержке фронтенд-приложение для вашего Trivia API на Go.

