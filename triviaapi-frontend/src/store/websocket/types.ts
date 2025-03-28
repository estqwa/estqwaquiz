// Типы для WebSocket Slice
import { WebSocketMessage } from '../../types/websocket';

/**
 * Состояние WebSocket для Redux-слайса
 */
export interface WebSocketState {
  isConnected: boolean; // Флаг активного соединения
  isConnecting: boolean; // Флаг процесса подключения
  reconnectAttempts: number; // Количество попыток переподключения
  lastConnectedAt: string | null; // Время последнего успешного подключения
  lastDisconnectedAt: string | null; // Время последнего отключения
  error: string | null; // Ошибка соединения
  lastMessage: WebSocketMessage | null; // Последнее полученное сообщение
  messageQueue: WebSocketMessage[]; // Очередь сообщений для отправки при переподключении
  shardId: number | null; // ID текущего шарда при использовании шардинга
} 