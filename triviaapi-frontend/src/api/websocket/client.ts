import { store } from '../../store';
import { 
  wsConnecting, 
  wsConnected, 
  wsDisconnected, 
  wsError,
  wsMessage 
} from '../../store/websocket/slice';
import { tokenRefreshed, logoutSuccess } from '../../store/auth/slice';
import { authService } from '../services/authService';
import { WebSocketMessage as ReduxWebSocketMessage, WebSocketMessage, MessagePriority } from '../../types/websocket';
import { transformKeysToSnakeCase, transformKeysToCamelCase } from '../../utils/api';

// Эта функция больше не нужна, так как мы будем получать тикет отдельным запросом
// const getWebSocketToken = (): string | null => {
//   const { token, csrfToken, useCookieAuth, isAuthenticated } = store.getState().auth;
//   
//   // Для WebSocket всегда используем Access Token (JWT) для аутентификации
//   // В режиме Cookie-based auth, Access Token в HttpOnly cookie, а в store хранится только csrfToken
//   if (useCookieAuth) {
//     // В режиме Cookie-based auth мы не имеем доступа к access_token из JS
//     // Нам нужен специальный эндпоинт для получения токена для WS
//     return null; // Вернем null, чтобы connect() вызвал refreshToken
//   } else {
//     return token; // В режиме Bearer Token возвращаем access_token из store
//   }
// };

export class WebSocketClient {
  private static instance: WebSocketClient;
  private socket: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private missedHeartbeats = 0;
  private messageHandlers = new Map<string, Array<(data: unknown) => void>>();
  private connectionPromise: Promise<void> | null = null;
  private isConnecting = false;

  private constructor() {}

  public static getInstance(): WebSocketClient {
    if (!WebSocketClient.instance) {
      WebSocketClient.instance = new WebSocketClient();
    }
    return WebSocketClient.instance;
  }

  // Подключение к WebSocket серверу с использованием WS-тикета
  public connect(ticket?: string | null): Promise<void> {
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

    this.connectionPromise = new Promise<void>(async (resolve, reject) => {
      try {
        // Проверяем, что у нас есть тикет для подключения
        if (!ticket) {
          // Если пользователь не аутентифицирован или тикет не предоставлен
          if (!store.getState().auth.isAuthenticated) {
            console.error('WebSocket connection cancelled: User not authenticated.');
            this.isConnecting = false;
            reject(new Error('Authentication required'));
            return;
          }

          // Если тикет не предоставлен, но пользователь аутентифицирован в режиме Bearer
          if (!store.getState().auth.useCookieAuth) {
            // Для режима Bearer продолжаем использовать access_token из store
            const storeToken = store.getState().auth.token;
            ticket = storeToken;
            
            // Если токена нет, пробуем его обновить
            if (!ticket) {
              console.warn("No access token for WebSocket, trying to refresh...");
              try {
                const { accessToken } = await authService.refreshToken();
                store.dispatch(tokenRefreshed({ token: accessToken }));
                ticket = accessToken;
              } catch (refreshError) {
                console.error("Failed to refresh token for WebSocket:", refreshError);
                store.dispatch(logoutSuccess()); // Выход, если не можем получить токен
                this.isConnecting = false;
                reject(new Error('Authentication required for WebSocket'));
                return;
              }
            }
          } else {
            // В режиме Cookie Auth требуется явный WS-тикет
            console.error('WebSocket connection requires a ticket in Cookie-Auth mode.');
            this.isConnecting = false;
            reject(new Error('WebSocket ticket required'));
            return;
          }
        }

        // URL для WebSocket подключения (с безопасной fallback на localhost)
        const wsBaseUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws'; // Fallback URL
        const wsUrl = `${wsBaseUrl}?token=${ticket}`;
        
        // Безопасное логирование URL без токена
        console.log('Connecting to WebSocket:', wsUrl.replace(ticket || '', '***')); // Не логируем токен/тикет

        // Закрываем старое соединение, если оно есть
        this.disconnectInternal(true); // true - не пытаться переподключаться

        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
          console.log('WebSocket connection established');
          this.reconnectAttempts = 0; // Сбрасываем счетчик попыток
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
          reject(new Error(errorMsg));
          // Не вызываем handleReconnection здесь, ждем onclose
        };

        this.socket.onmessage = (event) => {
          try {
            const rawMessage = JSON.parse(event.data as string) as WebSocketMessage;
            
            // Преобразуем данные сообщения из snake_case в camelCase
            const message: WebSocketMessage = {
              type: rawMessage.type,
              data: transformKeysToCamelCase(rawMessage.data),
              priority: rawMessage.priority
            };

            // Обработка heartbeat от сервера
            if (message.type === 'server:heartbeat') {
              this.missedHeartbeats = 0;
              return;
            }

            // Показываем список зарегистрированных типов при получении сообщения для отладки
            console.log('Получено сообщение типа:', message.type);
            console.log('Зарегистрированные типы:', Array.from(this.messageHandlers.keys()));
            console.log('Есть обработчик для этого типа:', this.messageHandlers.has(message.type));

            // Диспатч сообщения в Redux store (для логгирования или общей обработки)
            // Преобразуем внутреннее сообщение в формат ReduxWebSocketMessage
            store.dispatch(wsMessage({
              type: message.type,
              data: message.data
            } as ReduxWebSocketMessage));

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
              console.warn(`No WebSocket message handlers registered for type: ${message.type}`);
            }
          } catch (error: unknown) {
            console.error('Error parsing WebSocket message:', error, event.data);
            const errorMessage = error instanceof Error ? error.message : 'Unknown error';
            store.dispatch(wsError({ error: `Failed to parse WebSocket message: ${errorMessage}` }));
          }
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    }).finally(() => {
      // Очищаем промис после его завершения (успех или ошибка)
      // this.connectionPromise = null; // Не очищаем здесь, чтобы повторные вызовы connect возвращали результат
      this.isConnecting = false; // Убеждаемся, что флаг сброшен
    });

    return this.connectionPromise;
  }

  // Модифицируем handleAuthError, чтобы использовать механизм тикетов
  private async handleAuthError(reject: (reason?: any) => void): Promise<void> {
    try {
      // Для режима Cookie Auth мы не можем использовать обновленный access_token
      // Нужно полагаться на фронтенд-приложение для повторного получения тикета
      if (store.getState().auth.useCookieAuth) {
        console.error('WebSocket authentication failed. Need to request new WS-ticket.');
        reject(new Error('WebSocket authentication failed. Need to request new WS-ticket.'));
        return;
      }

      // Для режима Bearer обновляем токен как обычно
      const { accessToken } = await authService.refreshToken();
      store.dispatch(tokenRefreshed({ token: accessToken }));
      console.log('Token refreshed successfully after WS auth error. Reconnecting WebSocket...');
      // Вызываем connect() с новым токеном
      if (accessToken) {  // Проверяем, что accessToken не null
        this.connect(accessToken)
          .then(() => console.log('Successfully reconnected WebSocket with new token'))
          .catch(error => {
            console.error('Failed to reconnect after token refresh:', error);
            reject(error);
          });
      } else {
        console.error('Failed to reconnect: No access token after refresh');
        reject(new Error('No access token after refresh'));
      }
    } catch (refreshError) {
      console.error('Failed to refresh token after WS auth error:', refreshError);
      store.dispatch(logoutSuccess()); // Выход, если не удалось обновить токен
      reject(new Error('WebSocket authentication failed and token refresh failed')); // Отклоняем промис подключения
    }
  }

  // Обработка переподключения с экспоненциальной задержкой
  private handleReconnection(reject: (reason?: any) => void): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error(`Max reconnect attempts (${this.maxReconnectAttempts}) reached. Stopping reconnection.`);
      reject(new Error(`Failed to reconnect after ${this.maxReconnectAttempts} attempts`));
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts - 1), 30000); // Экспоненциальная задержка, макс 30 сек
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

    this.reconnectTimeout = setTimeout(() => {
      console.log(`Reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
      // Пытаемся переподключиться
      this.connect()
        .then(() => {
          console.log(`Successfully reconnected WebSocket after attempt ${this.reconnectAttempts}`);
          // TODO: Sync state on reconnect - Request current quiz state from backend here (via WS message or API call) to ensure consistency after connection loss.
        })
        .catch(error => {
          console.error(`Reconnect attempt ${this.reconnectAttempts} failed:`, error);
          // Явно не вызываем handleReconnection отсюда, т.к. это сделает onclose
        });
    }, delay);
  }

  // Отправка сообщения через WebSocket
  public sendMessage(type: string, data: any, priority: MessagePriority = MessagePriority.NORMAL): boolean {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error(`Cannot send message, WebSocket not connected, type: ${type}`);
      return false;
    }
    
    try {
      // Преобразуем данные из camelCase в snake_case перед отправкой
      const transformedData = transformKeysToSnakeCase(data);
      const message: WebSocketMessage = { type, data: transformedData, priority };
      this.socket.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('Error sending WebSocket message:', error);
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
  public addMessageHandler<T>(type: string, handler: (data: T) => void): () => void {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, []);
    }
    
    const handlers = this.messageHandlers.get(type)!;
    handlers.push(handler as (data: unknown) => void);
    console.log(`Зарегистрирован WebSocket обработчик для типа: "${type}", всего: ${handlers.length}`); // Расширенное логирование
    
    // Выводим все текущие типы сообщений с обработчиками для отладки
    console.log('Текущие типы сообщений с обработчиками:', Array.from(this.messageHandlers.keys()));
    
    // Возвращаем функцию для удаления этого обработчика
    return () => {
      this.removeMessageHandler(type, handler as (data: unknown) => void);
    };
  }

  // Удаление обработчика
  public removeMessageHandler(type: string, handler: (data: unknown) => void): void {
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
    this.missedHeartbeats = 0;
  }
}

// Экспортируем синглтон клиента
export const wsClient = WebSocketClient.getInstance(); 